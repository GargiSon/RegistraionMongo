package mongo

import (
	"context"
	"go2/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func InsertResetToken(ctx context.Context, userID primitive.ObjectID, tokenHash string, expiry int64) error {
	_, err := GetCollection("RegistrationMongo", "password_reset_tokens").
		InsertOne(ctx, bson.M{
			"user_id":      userID,
			"token":        tokenHash,
			"token_expiry": expiry,
		})
	return err
}

func FindResetToken(ctx context.Context, tokenHash string) (model.PasswordResetToken, error) {
	var token model.PasswordResetToken
	err := GetCollection("RegistrationMongo", "password_reset_tokens").
		FindOne(ctx, bson.M{"token": tokenHash}).Decode(&token)
	return token, err
}

func DeleteResetTokensByUserID(ctx context.Context, userID primitive.ObjectID) error {
	_, err := GetCollection("RegistrationMongo", "password_reset_tokens").
		DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
