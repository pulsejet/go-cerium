package main_test

import (
	"fmt"
	"bytes"
	"testing"
	"net/http"
	"net/http/httptest"
	"os"
	"encoding/json"

	c "github.com/pulsejet/go-cerium/controllers"
	u "github.com/pulsejet/go-cerium/utils"
	"github.com/pulsejet/go-cerium/models"

	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
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
	err := godotenv.Load()
	rno = os.Getenv("TEST_ROLL")
	if err != nil {
		fmt.Errorf("Error loading .env file")
		os.Exit(1)
	}
	
	// Create dummy profile
	profile := ProfileResponse{}
	profile.Id=1234 
	profile.FirstName="Test" 
	profile.LastName="Subject"; 
	profile.RollNumber=rno;
	profile.ProfilePicture="";
	profile.Email="test@gmail.com";

	rno := profile.RollNumber
	collection := u.Collection("users")
	var ropts options.ReplaceOptions
	ropts.SetUpsert(true)

	// Insert dummy profile into database
	_, err = collection.ReplaceOne(
		u.Context(), bson.M{"rollnumber": rno}, profile, &ropts)

	if err != nil {
		fmt.Println("Could not complete setup")
		os.Exit(1)
	}
}

func shutdown() {
	// Remove the dummy profile created
	collection := u.Collection("users")
	_, err := collection.DeleteOne(u.Context(), bson.M{"rollnumber": rno})
	if err != nil {
		fmt.Printf("remove fail %v\n", err)
	}

	// Cleanup any form or response created by test subject
	collection = u.Collection("forms")
	_, err = collection.DeleteMany(u.Context(), bson.M{"creator": rno})
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

	collection := u.Collection("users")
	err := collection.FindOne(u.Context(), bson.M{"rollnumber": rno}).Decode(&user)
	checkError(err, t)
}

// Tests creation of new form by user
func TestNewFormCreate(t *testing.T) {
	// Create dummy form
	form := createDummyForm()

	tempR := httptest.NewRecorder()
	c.SetCookie(tempR, rno)

	handler := http.HandlerFunc(c.CreateForm)
	
	formJson, err := json.Marshal(form)

	req, err := http.NewRequest("POST", "/api/form", bytes.NewBuffer(formJson))
	checkError(err, t)
	
	req.Header.Add("Cookie", tempR.HeaderMap["Set-Cookie"][0])
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	// Fire the request
	handler.ServeHTTP(rr, req)

	//Confirm the response has the right status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Status code differs. Expected %d .\n Got %d instead", http.StatusOK, status)
	}

	dbForm := &models.Form{}
	err = u.Collection("forms").FindOne(u.Context(), bson.M{"creator":rno}).Decode(&dbForm)
	if dbForm.Name != "Test Form" {
		t.Errorf("Form in db different from one created in test")
	}
}

// Tests empty form are not created, and status 400 is sent
func TestEmptyForm(t *testing.T) {
	// Create dummy form
	form := createDummyForm()

	tempR := httptest.NewRecorder()
	c.SetCookie(tempR, rno)

	handler := http.HandlerFunc(c.CreateForm)

	// Empty the pages and create request
	form.Pages = []models.Page{}
	formJson, _ := json.Marshal(form)
	request, _ := http.NewRequest("POST", "/api/form", bytes.NewBuffer(formJson))
	request.Header.Add("Cookie", tempR.HeaderMap["Set-Cookie"][0])
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	
	handler.ServeHTTP(recorder, request)

	//Confirm the response has the right status code
	if status := recorder.Code; status != http.StatusBadRequest {
		t.Errorf("Status code differs. Expected %d Got %d instead", http.StatusBadRequest, status)
	}
}

func checkError(err error, t *testing.T) {
	if err != nil {
		t.Errorf("An error occurred. %v", err)
	}
}

func createDummyForm() models.Form {
	widget := &models.Widget{
		Type  :"short_answer",
		Props :map[string]interface{}{"question" : "Question first","validators" : "{required : false}" },
	}
	page := &models.Page{
		Title       :"Test Form",
		Description :"",
	}
	page.Widgets = []models.Widget{*widget}
	form := &models.Form{
		CanEdit        :true,
		Pages          :[]models.Page{*page},
		RequireLogin   :true,
		CollectEmail   :false,
		SingleResponse :true,
	}
	return *form
}
