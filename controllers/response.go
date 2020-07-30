package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/pulsejet/go-cerium/models"
	u "github.com/pulsejet/go-cerium/utils"
)

// ResponsesRequest : helper for post processing
type ResponsesRequest struct {
	Type string `json:"type"`
}

// CreateResponse : API handler for POST-ing new response
var CreateResponse = func(w http.ResponseWriter, r *http.Request) {
	formid := mux.Vars(r)["formid"]

	// Check if login is required
	rno := GetRollNo(w, r, false)
	collection := u.Collection(r.Context(), "forms")
	objID, _ := primitive.ObjectIDFromHex(formid)
	filt := bson.M{"_id": objID}
	form := &models.Form{}
	collection.FindOne(r.Context(), filt).Decode(form)
	if form.RequireLogin && rno == "" {
		u.Respond(w, u.Message(false, "Not Found"), 401)
		return
	}

	// Return error if form is closed
	year2000 := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if form.IsClosed || time.Now().UTC().After(form.CloseOn) && year2000.Before(form.CloseOn) {
		u.Respond(w, u.Message(false, "Form Closed"), 404)
		return
	}

	// Save the response
	response := &models.FormResponse{}
	err := json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Fill in the responses
	response.FormID = formid
	response.Timestamp = time.Now()
	response.Responses["timestamp"] = response.Timestamp
	if form.CollectEmail {
		response.Filler = rno
		response.Responses["filler"] = response.Filler
	}

	// Check if form already filled for single response
	if form.SingleResponse && HasFilledAnon(r.Context(), formid, rno) {
		u.Respond(w, u.Message(false, "User has already filled this form"), 403)
		return
	}

	// Create an anon filler object
	if form.RequireLogin {
		anonResponse := &models.FormAnonResponder{}
		anonResponse.Filler = rno
		anonResponse.FormID = formid

		// Add the anon filler to fillers collection
		collection = u.Collection(r.Context(), "filler")
		collection.InsertOne(r.Context(), anonResponse)
	}

	// Add the document to the responses collection
	collection = u.Collection(r.Context(), "responses")
	cur, _ := collection.InsertOne(r.Context(), response)
	id := cur.InsertedID

	// Log to console
	log.Debug(rno, ": new response for form", formid)

	u.Respond(w, map[string]interface{}{"id": id}, 200)
}

// GetResponses : API handler for getting JSON responses (for CSV)
var GetResponses = func(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	rno := GetRollNo(w, r, true)
	if rno == "" {
		return
	}

	id := mux.Vars(r)["formid"]
	data := strings.Split(id, "-")
	formid := data[0]
	token := data[1]

	// Check privileges
	form := &models.Form{}
	collection := u.Collection(r.Context(), "forms")
	objID, _ := primitive.ObjectIDFromHex(formid)
	filt := bson.M{"$and": bson.A{
		bson.M{"_id": objID},
		bson.M{"responsetoken": token}}}
	err := collection.FindOne(r.Context(), filt).Decode(&form)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Get responses
	responses := []*models.FormResponse{}
	collection = u.Collection(r.Context(), "responses")
	cur, err := collection.Find(r.Context(), bson.M{"formid": formid})
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Iterate and collect responses
	for cur.Next(context.TODO()) {
		var elem models.FormResponse
		err := cur.Decode(&elem)
		if err != nil {
			log.Error(err)
		}
		responses = append(responses, &elem)
	}

	// Postprocess if wanted
	rq := &ResponsesRequest{}
	json.NewDecoder(r.Body).Decode(rq)
	if rq.Type == "array" {
		u.Respond(w, arrayResponse(form, responses), 200)
		return
	}

	u.Respond(w, responses, 200)
}

func arrayResponse(f *models.Form, r []*models.FormResponse) [][]string {
	// Get form fields
	fields, fnames := formFields(f)

	// Make grand array of arrays
	a := make([][]string, len(r)+1)

	// Construct header
	a[0] = make([]string, len(fields))
	for j := range fields {
		a[0][j] = fnames[fields[j]]
	}

	// Iterate each response
	for iw := range r {
		i := iw + 1
		a[i] = make([]string, len(fields))

		for j := range fields {
			cf := r[iw].Responses[fields[j]]
			switch cf.(type) {
			default:
				a[i][j] = fmt.Sprintf("%s", cf)
			case float32, float64:
				a[i][j] = fmt.Sprintf("%9.f", cf)
			case nil:
				a[i][j] = ""
			case primitive.DateTime:
				a[i][j] = primitiveToTime(cf.(primitive.DateTime)).String()
			}
		}
	}

	return a
}

func formFields(f *models.Form) ([]string, map[string]string) {
	// Initialize
	m := map[string]string{}
	a := make([]string, 0)

	// Add extra fields
	a = append(a, "timestamp")
	m["timestamp"] = "Timestamp"

	if f.CollectEmail {
		a = append(a, "filler")
		m["filler"] = "Filler"
	}

	// Construct fields
	for pi := range f.Pages {
		for wi := range f.Pages[pi].Widgets {
			m[f.Pages[pi].Widgets[wi].UID] = f.Pages[pi].Widgets[wi].Props["question"].(string)
			a = append(a, f.Pages[pi].Widgets[wi].UID)
		}
	}

	return a, m
}

// Time returns the date as a time type.
func primitiveToTime(d primitive.DateTime) time.Time {
	return time.Unix(int64(d)/1000, int64(d)%1000*1000000)
}

// HasFilledAnon returns true if the person has already filled this form
func HasFilledAnon(ctx context.Context, formid string, filler string) bool {
	// Filter matching form id and filler
	collection := u.Collection(ctx, "filler")
	anonExist := collection.FindOne(ctx, bson.M{
		"$and": bson.A{
			bson.M{"formid": formid},
			bson.M{"filler": filler}}})

	// Try to get the object
	anonObj := &models.FormAnonResponder{}
	err := anonExist.Decode(anonObj)
	return err == nil
}
