package mongo

import (
	"context"
	"go2/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetPaginatedUsers(ctx context.Context, page, limit int, sortField, sortOrder string) ([]model.User, int64, error) {
	offset := (page - 1) * limit

	findOptions := options.Find().
		SetSkip(int64(offset)).
		SetLimit(int64(limit)).
		SetSort(bson.D{{Key: sortField, Value: getSortOrderValue(sortOrder)}})

	cursor, err := GetUserCollection().Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []model.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	total, err := GetUserCollection().CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func getSortOrderValue(order string) int {
	if order == "asc" {
		return 1
	}
	return -1
}
