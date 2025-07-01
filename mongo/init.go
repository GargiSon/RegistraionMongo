package mongo

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func InitMongoData() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db := os.Getenv("MONGO_DB_NAME")
	if db == "" {
		log.Println("MONGO_DB_NAME not set in environment variables")
		return
	}

	countryColl := GetCollection(db, "countries")
	countryCount, err := countryColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		log.Println("Error checking countries:", err)
	} else if countryCount == 0 {
		countries := []any{
			bson.M{"name": "INDIA"},
			bson.M{"name": "AFGHANISTHAN"},
			bson.M{"name": "FRANCE"},
		}
		if _, err := countryColl.InsertMany(ctx, countries); err != nil {
			log.Println("Failed to insert default countries:", err)
		} else {
			fmt.Println("Inserted default countries.")
		}
	} else {
		fmt.Println("Countries already exist.")
	}

	//Admins manually insert in database, and forgot password is used for reseting the password
	// adminColl := GetCollection(db, "admins")
	// adminCount, err := adminColl.CountDocuments(ctx, bson.M{})
	// if err != nil {
	// 	log.Println("Error checking admins:", err)
	// 	return
	// }
	// if adminCount == 0 {
	// 	adminEmail := os.Getenv("ADMIN_EMAIL")
	// 	adminPassword := os.Getenv("ADMIN_PASSWORD")

	// 	if adminEmail == "" || adminPassword == "" {
	// 		log.Println("ADMIN_EMAIL or ADMIN_PASSWORD not set")
	// 		return
	// 	}

	// 	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	// 	if err != nil {
	// 		log.Println("Failed to hash admin password:", err)
	// 		return
	// 	}

	// 	admin := bson.M{
	// 		"email":    adminEmail,
	// 		"password": string(hashedPassword),
	// 	}
	// 	if _, err := adminColl.InsertOne(ctx, admin); err != nil {
	// 		log.Println("Failed to insert default admin:", err)
	// 	} else {
	// 		fmt.Println("Inserted default admin.")
	// 	}
	// }
}
