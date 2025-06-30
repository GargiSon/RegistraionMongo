package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func GetAdminByEmail(email string) (Admin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var admin Admin
	collection := GetCollection("RegistrationMongo", "AdminNew")

	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&admin)
	return admin, err
}

func CheckAdminExists(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("RegistrationMongo", "AdminNew")
	count, err := collection.CountDocuments(ctx, bson.M{"email": email})
	return count > 0, err
}

func UpdateAdminPassword(email string, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection("RegistrationMongo", "AdminNew")
	_, err := collection.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"password": hashedPassword}},
	)
	return err
}
