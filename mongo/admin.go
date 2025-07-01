package mongo

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func getDBName() string {
	return os.Getenv("MONGO_DB_NAME")
}

func GetAdminByEmail(email string) (Admin, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var admin Admin
	collection := GetCollection(getDBName(), "admins")

	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&admin)
	return admin, err
}

func CheckAdminExists(email string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection(getDBName(), "admins")
	count, err := collection.CountDocuments(ctx, bson.M{"email": email})
	return count > 0, err
}

func UpdateAdminPassword(email string, hashedPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := GetCollection(getDBName(), "admins")
	_, err := collection.UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"password": hashedPassword}},
	)
	return err
}
