package mongo

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	clientInstance *mongo.Client
	clientOnce     sync.Once
)

func Connect() *mongo.Client {
	clientOnce.Do(func() {
		// Find env
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading .env file: %s", err)
		}

		// Get value from env
		MONGO_URI := os.Getenv("MONGO_URI")

		// Connect to database
		clientOptions := options.Client().ApplyURI(MONGO_URI)
		client, err := mongo.Connect(context.Background(), clientOptions)
		if err != nil {
			log.Fatalf("Failed to connect to MongoDB: %v", err)
		}

		// Check the connection
		if err := client.Ping(context.Background(), nil); err != nil {
			log.Fatalf("MongoDB not responding: %v", err)
		}
		fmt.Println("Connected to MongoDB!!")
		clientInstance = client
	})
	return clientInstance
}

func GetCollection(dbName, collectionName string) *mongo.Collection {
	client := Connect()
	return client.Database(dbName).Collection(collectionName)
}
