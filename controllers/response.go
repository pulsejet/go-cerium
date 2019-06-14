package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/pulsejet/cerium/models"
	u "github.com/pulsejet/cerium/utils"
)

var CreateResponse = func(w http.ResponseWriter, r *http.Request) {
	formid := mux.Vars(r)["formid"]

	// Check if login is required
	rno := GetRollNo(w, r, false)
	collection := u.Collection("forms")
	objID, _ := primitive.ObjectIDFromHex(formid)
	filt := bson.M{"_id": objID}
	form := &models.Form{}
	collection.FindOne(u.Context(), filt).Decode(form)
	if form.RequireLogin && rno == "" {
		u.Respond(w, u.Message(false, "Not Found"), 401)
		return
	}

	// Save the response
	response := &models.FormResponse{}
	err := json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	response.FormId = formid
	response.Timestamp = time.Now()
	if form.CollectEmail {
		response.Filler = rno
	}

	collection = u.Collection("responses")
	cur, _ := collection.InsertOne(u.Context(), response)
	id := cur.InsertedID

	u.Respond(w, map[string]interface{}{"id": id}, 200)
}

var GetResponses = func(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	rno := GetRollNo(w, r, true)
	if rno == "" {
		return
	}

	formid := mux.Vars(r)["formid"]

	// Check privileges
	collection := u.Collection("forms")
	objID, _ := primitive.ObjectIDFromHex(formid)
	filt := bson.M{"$and": bson.A{
		bson.M{"_id": objID},
		bson.M{"creator": rno}}}
	var fopts options.CountOptions
	fopts.SetLimit(1)
	c, _ := collection.CountDocuments(u.Context(), filt, &fopts)
	if c <= 0 {
		u.Respond(w, u.Message(false, "Not Found"), 404)
		return
	}

	// Get responses
	responses := []*models.FormResponse{}
	collection = u.Collection("responses")
	cur, err := collection.Find(u.Context(), bson.M{"formid": formid})
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	for cur.Next(context.TODO()) {
		var elem models.FormResponse
		err := cur.Decode(&elem)
		if err != nil {
			log.Println(err)
		}
		responses = append(responses, &elem)
	}

	u.Respond(w, responses, 200)
}
