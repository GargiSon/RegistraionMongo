package mongo

import (
	"context"
	"go2/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetUserCollection() *mongo.Collection {
	return GetCollection("RegistrationMongo", "users")
}

func EmailExists(ctx context.Context, email string) bool {
	count, _ := GetUserCollection().CountDocuments(ctx, bson.M{"email": email})
	return count > 0
}

func MobileExists(ctx context.Context, mobile string) bool {
	count, _ := GetUserCollection().CountDocuments(ctx, bson.M{"mobile": mobile})
	return count > 0
}

func InsertUser(ctx context.Context, user model.User) error {
	_, err := GetUserCollection().InsertOne(ctx, user)
	return err
}

func FindUserByID(ctx context.Context, id primitive.ObjectID) (model.User, error) {
	var user model.User
	err := GetUserCollection().FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	return user, err
}

func UpdateUserByID(ctx context.Context, id primitive.ObjectID, update bson.M) error {
	_, err := GetUserCollection().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

func DeleteUserByID(ctx context.Context, id primitive.ObjectID) error {
	_, err := GetUserCollection().DeleteOne(ctx, bson.M{"_id": id})
	return err
}
