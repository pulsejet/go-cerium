package controllers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/gorilla/mux"

	"github.com/pulsejet/go-cerium/models"
	u "github.com/pulsejet/go-cerium/utils"
)

var CreateForm = func(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	rno := GetRollNo(w, r, true)
	if rno == "" {
		return
	}

	// Decode the JSON form
	form := &models.Form{}
	err := json.NewDecoder(r.Body).Decode(form)
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Forms without pages are not valid
	if len(form.Pages) == 0 {
		u.Respond(w, u.Message(false, "No Pages"), 400)
		return
	}

	// Setup fields
	assignUids(form)
	form.Name = form.Pages[0].Title
	responseToken := u.RandSeq(50)
	form.ResponseToken = responseToken
	collection := u.Collection("forms")

	// Update or create new
	var id interface{}
	if r.Method == "PUT" {
		cid := mux.Vars(r)["id"]
		objID, _ := primitive.ObjectIDFromHex(cid)
		filt := bson.M{"$and": bson.A{
			bson.M{"_id": objID},
			bson.M{"creator": rno}}}

		var res *mongo.UpdateResult
		res, err = collection.ReplaceOne(u.Context(), filt, form)
		id = cid

		if res.MatchedCount == 0 {
			u.Respond(w, u.Message(false, "Not Found"), 404)
			return
		}
	} else {
		form.Creator = rno
		var res *mongo.InsertOneResult
		res, err = collection.InsertOne(u.Context(), form)
		id = res.InsertedID
	}

	// Check for errors and return form id
	if err != nil {
		log.Println(err)
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Log to console
	log.Println(rno, ": new form", id)

	u.Respond(w, map[string]interface{}{"id": id, "token": responseToken}, 200)
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

	// Get the form
	collection := u.Collection("forms")
	objID, _ := primitive.ObjectIDFromHex(id)
	err := collection.FindOne(u.Context(), bson.M{"_id": objID}).Decode(&form)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Check if editable
	rno := GetRollNo(w, r, false)
	if rno != "" {
		form.CanEdit = rno == form.Creator
	}

	// Check if already filled
	if !form.CanEdit && form.SingleResponse && HasFilledAnon(id, rno) {
		u.Respond(w, u.Message(false, "User has already filled this form"), 403)
		return
	}

	// Login required
	if form.RequireLogin && rno == "" {
		u.Respond(w, u.Message(false, "Unauthorized: Please login to continue"), 401)
		return
	}

	u.Respond(w, form, 200)
}

var GetAllForms = func(w http.ResponseWriter, r *http.Request) {
	// Get roll number
	rno := GetRollNo(w, r, false)

	// To extrach data out of collection.Find()
	type formDB struct {
		ID    primitive.ObjectID `bson:"_id"`
		Name  string             `bson:"name"`
		Token string             `bson:"responsetoken"`
	}

	// To send data to frontend
	type formDetails struct {
		ID    string
		Name  string
		Token string
	}

	var forms []formDetails

	collection := u.Collection("forms")
	count, err := collection.CountDocuments(u.Context(), bson.M{"creator": rno})
	if count == 0 {
		u.Respond(w, forms, 200)
	}

	// To Set which fields are required in the output
	type fields struct {
		ID    int `bson:"_id"`
		Name  int `bson:"name"`
		Token int `bson:"responsetoken"`
	}
	projection := fields{ID: 1, Name: 1, Token: 1}
	opt := &options.FindOptions{}
	opt.SetProjection(projection)

	// Get all form ids for this roll number
	values, err := collection.Find(u.Context(), bson.M{"creator": rno}, opt)
	if err != nil {
		u.Respond(w, u.Message(false, err.Error()), 400)
		return
	}

	// Iterate and collect responses
	for values.Next(context.TODO()) {
		var elem formDB
		err := values.Decode(&elem)
		if err != nil {
			log.Println(err)
		}
		forms = append(forms, formDetails{ID: (&elem).ID.Hex(), Name: (&elem).Name, Token: (&elem).Token})
	}
	log.Println("all forms created by ", rno, " sent")
	u.Respond(w, forms, 200)
}
