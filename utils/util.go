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

func Message(status bool, message string) map[string]interface{} {
	return map[string]interface{}{"status": status, "message": message}
}

func Respond(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Database(ctx context.Context) *mongo.Database {
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("CONNECTION")))
	err = client.Connect(ctx)
	if err != nil {
		fmt.Print(err)
		return nil
	}

	return client.Database(os.Getenv("DATABASE"))
}

func Collection(ctx context.Context, name string) *mongo.Collection {
	return Database(ctx).Collection(name)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func RandSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func RandomId() string {
	return RandSeq(24)
}
