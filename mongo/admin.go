package mongo

import (
	"context"
	"go2/model"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func getDBName() string {
	return os.Getenv("MONGO_DB_NAME")
}

func GetAdminByEmail(ctx context.Context, email string) (model.Admin, error) {
	var admin model.Admin
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

func UpdateAdminPassword(ctx context.Context, adminID primitive.ObjectID, hashedPassword string) error {
	_, err := GetCollection(getDBName(), "admins").
		UpdateByID(ctx, adminID, bson.M{
			"$set": bson.M{"password": hashedPassword},
		})
	return err
}
