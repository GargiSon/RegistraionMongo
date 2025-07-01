package utils

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"go2/mongo"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func SetFlashMessage(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "flash",
		Value: message,
		Path:  "/",
	})
}

func GetFlashMessage(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("flash")
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "flash",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return cookie.Value
}

func GetCountriesFromDB() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := mongo.GetCollection("RegistrationMongo", "countries")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var countries []string
	for cursor.Next(ctx) {
		var doc struct {
			Name string `bson:"name"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		countries = append(countries, doc.Name)
	}
	return countries, nil
}

func GenerateSecureToken(length int) string {
	bytes := make([]byte, length)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func HashSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
