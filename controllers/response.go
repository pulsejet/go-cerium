package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/pulsejet/cerium/models"
	u "github.com/pulsejet/cerium/utils"
)

var CreateResponse = func(w http.ResponseWriter, r *http.Request) {
	formid := mux.Vars(r)["formid"]

	response := &models.FormResponse{}
	err := json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	response.FormId = formid
	response.Timestamp = time.Now()

	collection := u.Collection("responses")
	res, err := collection.InsertOne(u.Context(), response)
	id := res.InsertedID

	u.Respond(w, map[string]interface{}{"id": id}, 200)
}

var GetResponses = func(w http.ResponseWriter, r *http.Request) {
	formid := mux.Vars(r)["formid"]

	responses := []*models.FormResponse{}

	collection := u.Collection("responses")
	cur, err := collection.Find(u.Context(), bson.M{"formid": formid})
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	for cur.Next(context.TODO()) {
		var elem models.FormResponse
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		responses = append(responses, &elem)
	}

	u.Respond(w, responses, 200)
}
