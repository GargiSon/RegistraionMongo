package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func InitMongoData() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := "RegistrationMongo"

	countryColl := GetCollection(db, "Countries")
	countryCount, err := countryColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println("Error checking Countries:", err)
	} else if countryCount == 0 {
		countries := []interface{}{
			bson.M{"name": "INDIA"},
			bson.M{"name": "AFGHANISTHAN"},
			bson.M{"name": "FRANCE"},
		}
		_, err := countryColl.InsertMany(ctx, countries)
		if err != nil {
			log.Println("Failed to insert default countries:", err)
		} else {
			fmt.Println("Inserted default countries.")
		}
	} else {
		fmt.Println("Countries already exist.")
	}

	adminColl := GetCollection(db, "AdminNew")
	adminCount, err := adminColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println("Error checking AdminNew:", err)
	} else if adminCount == 0 {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin1001"), bcrypt.DefaultCost)
		admin := bson.M{
			"email":    "gargi.soni@loginradius.com",
			"password": string(hashedPassword),
		}
		_, err := adminColl.InsertOne(ctx, admin)
		if err != nil {
			log.Println("Failed to insert default admin:", err)
		} else {
			fmt.Println("Inserted default admin.")
		}
	} else {
		fmt.Println("Admin already exists.")
	}

	newColl := GetCollection(db, "New")
	docCount, err := newColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println("Error checking New collection:", err)
	} else if docCount == 0 {
		fmt.Println("'New' collection ready for user registrations.")
	}
}
