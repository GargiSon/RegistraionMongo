package mongo

import (
	"context"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func setFlashMessage(w http.ResponseWriter, message string) {
	http.SetCookie(w, &http.Cookie{
		Name:  "flash",
		Value: message,
		Path:  "/",
	})
}

func getFlashMessage(w http.ResponseWriter, r *http.Request) string {
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

func getCountriesFromDB() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("RegistrationMongo", "Countries")

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
