package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gorilla/mux"

	"github.com/pulsejet/cerium/models"
	u "github.com/pulsejet/cerium/utils"
)

var CreateForm = func(w http.ResponseWriter, r *http.Request) {
	form := &models.Form{}
	err := json.NewDecoder(r.Body).Decode(form)
	if err != nil {
		log.Fatal(err)
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	assignUids(form)
	collection := u.Collection("forms")

	var id interface{}
	if r.Method == "PUT" {
		cid := mux.Vars(r)["id"]
		objID, _ := primitive.ObjectIDFromHex(cid)
		_, err = collection.ReplaceOne(u.Context(), bson.M{"_id": objID}, form)
		id = cid
	} else {
		var res *mongo.InsertOneResult
		res, err = collection.InsertOne(u.Context(), form)
		id = res.InsertedID
	}

	if err != nil {
		log.Fatal(err)
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	u.Respond(w, map[string]interface{}{"id": id}, 200)
}

/** Set random UID for each widget */
func assignUids(form *models.Form) {
	for i := 0; i < len(form.Pages); i++ {
		for j := 0; j < len(form.Pages[i].Widgets); j++ {
			w := &form.Pages[i].Widgets[j]
			if w.Uid == "" {
				w.Uid = u.RandomId()
			}
		}
	}
}

var GetForm = func(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	form := &models.Form{}

	collection := u.Collection("forms")
	objID, _ := primitive.ObjectIDFromHex(id)
	err := collection.FindOne(u.Context(), bson.M{"_id": objID}).Decode(&form)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	u.Respond(w, form, 200)
}
