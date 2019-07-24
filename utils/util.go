package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Message : a status message possibly representing an error
func Message(status bool, message string) map[string]interface{} {
	return map[string]interface{}{"status": status, "message": message}
}

// Responsd : respond with JSON data and status code
func Respond(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Database : Get pointer to database with context
func Database(ctx context.Context) *mongo.Database {
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("CONNECTION")))
	if err != nil {
		fmt.Print(err)
		return nil
	}
	err = client.Connect(ctx)
	if err != nil {
		fmt.Print(err)
		return nil
	}

	return client.Database(os.Getenv("DATABASE"))
}

// Collection : get pointer to collection with context
func Collection(ctx context.Context, name string) *mongo.Collection {
	return Database(ctx).Collection(name)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

// RandSeq : generate pseudorandom sequence of alphabets
func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RandomID : generate a 24-character pseudorandom sequence
func RandomID() string {
	return RandSeq(24)
}
