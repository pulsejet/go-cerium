package controllers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	c "github.com/pulsejet/go-cerium/controllers"
	"github.com/pulsejet/go-cerium/models"
	u "github.com/pulsejet/go-cerium/utils"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var rno string

type ProfileResponse struct {
	Id             int    `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	RollNumber     string `json:"roll_number"`
	ProfilePicture string `json:"profile_picture"`
	Email          string `json:"email"`
}

func setup() {
	err := godotenv.Load("../.env")
	rno = os.Getenv("TEST_ROLL")
	if err != nil {
		fmt.Errorf("Error loading .env file")
		os.Exit(1)
	}

	// Create dummy profile
	profile := ProfileResponse{}
	profile.Id = 1234
	profile.FirstName = "Test"
	profile.LastName = "Subject"
	profile.RollNumber = rno
	profile.ProfilePicture = ""
	profile.Email = "test@gmail.com"

	rno := profile.RollNumber
	collection := u.Collection(context.Background(), "users")
	var ropts options.ReplaceOptions
	ropts.SetUpsert(true)

	// Insert dummy profile into database
	_, err = collection.ReplaceOne(
		context.Background(), bson.M{"rollnumber": rno}, profile, &ropts)

	if err != nil {
		fmt.Println("Could not complete setup")
		os.Exit(1)
	}
}

func shutdown() {
	// Remove the dummy profile created
	collection := u.Collection(context.Background(), "users")
	_, err := collection.DeleteOne(context.Background(), bson.M{"rollnumber": rno})
	if err != nil {
		fmt.Printf("remove fail %v\n", err)
	}

	// Cleanup any form or response created by test subject
	collection = u.Collection(context.Background(), "forms")
	_, err = collection.DeleteMany(context.Background(), bson.M{"creator": rno})
	if err != nil {
		fmt.Printf("remove fail %v\n", err)
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

// Tests that the setup() has created db entry
func TestProfile(t *testing.T) {
	user := &ProfileResponse{}

	collection := u.Collection(context.Background(), "users")
	err := collection.FindOne(context.Background(), bson.M{"rollnumber": rno}).Decode(&user)
	checkError(err, t)
}

// Tests creation of new form by user
func TestNewFormCreate(t *testing.T) {
	// Create dummy form
	form := createDummyForm()

	handler := http.HandlerFunc(c.CreateForm)

	formJson, _ := json.Marshal(form)

	req := requestAPI("POST", "/api/form", formJson)
	rr := httptest.NewRecorder()

	// Fire the request
	handler.ServeHTTP(rr, req)

	//Confirm the response has the right status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Status code differs. Expected %d .\n Got %d instead", http.StatusOK, status)
	}

	dbForm := &models.Form{}
	err := u.Collection(context.Background(), "forms").FindOne(context.Background(), bson.M{"creator": rno}).Decode(&dbForm)
	checkError(err, t)
	if dbForm.Name != "Test Form" {
		t.Errorf("Form in db different from one created in test")
	}
}

// Tests empty form are not created, and status 400 is sent
func TestEmptyForm(t *testing.T) {
	// Create dummy form
	form := createDummyForm()

	handler := http.HandlerFunc(c.CreateForm)

	// Empty the pages and create request
	form.Pages = []models.Page{}
	formJson, _ := json.Marshal(form)
	request := requestAPI("POST", "/api/form", formJson)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	//Confirm the response has the right status code
	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("Status code differs. Expected %d Got %d instead", http.StatusBadRequest, status)
	}
}

// Tests for editing form
func TestEditForm(t *testing.T) {
	// Create dummy form
	form := createDummyForm()
	form.Pages[0].Title = "Edit Form"
	form.Name = "Edit Form"
	form.Creator = rno

	res, _ := u.Collection(context.Background(), "forms").InsertOne(context.Background(), form)
	id := res.InsertedID.(primitive.ObjectID)

	r := mux.NewRouter()
	r.HandleFunc("/api/form/{id}", c.CreateForm)

	// Empty the pages and create request
	form.Pages[0].Title = "Post Edit Form"
	formJson, _ := json.Marshal(form)
	request := requestAPI("PUT", "/api/form/"+id.Hex(), formJson)

	recorder := httptest.NewRecorder()

	r.ServeHTTP(recorder, request)

	//Confirm the response has the right status code
	if status := recorder.Code; status != http.StatusOK {
		t.Errorf("Status code differs. Expected %d Got %d instead", http.StatusOK, status)
	}

	// Make sure that form has been edited
	dbForm := &models.Form{}
	err := u.Collection(context.Background(), "forms").FindOne(context.Background(),
		bson.M{"$and": bson.A{bson.M{"_id": id}, bson.M{"creator": rno}}}).Decode(&dbForm)
	checkError(err, t)
	if dbForm.Name != "Post Edit Form" {
		t.Errorf("Form in db different from one edited in test")
	}
}

func requestAPI(Method string, API string, formString []byte) *http.Request {
	tempR := httptest.NewRecorder()
	c.SetCookie(tempR, rno)

	request, _ := http.NewRequest(Method, API, bytes.NewBuffer(formString))
	request.Header.Add("Cookie", tempR.HeaderMap["Set-Cookie"][0])
	request.Header.Set("Content-Type", "application/json")

	return request
}

func checkError(err error, t *testing.T) {
	if err != nil {
		t.Errorf("An error occurred. %v", err)
	}
}

func createDummyForm() models.Form {
	widget := &models.Widget{
		Type:  "short_answer",
		Props: map[string]interface{}{"question": "Question first", "validators": "{required : false}"},
	}
	page := &models.Page{
		Title:       "Test Form",
		Description: "",
	}
	page.Widgets = []models.Widget{*widget}
	form := &models.Form{
		CanEdit:        true,
		Pages:          []models.Page{*page},
		RequireLogin:   true,
		CollectEmail:   false,
		SingleResponse: true,
	}
	return *form
}
