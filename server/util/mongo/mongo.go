package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

// NewMongoDb initializes and returns a MongoDB database connection.
func NewMongoDb(uri, dbName string) *mongo.Database {
	if uri == "" {
		zap.S().Errorf("MongoDB uri is empty")
		// TODO: return error
	}
	if dbName == "" {
		// TODO: same as name
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		zap.S().Panicf("failed to create MongoDB client: %+v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		zap.S().Warnf("MongoDB ping warning (server will retry on demand): %+v", err)
	} else {
		zap.S().Info("Successfully connected to MongoDB")
	}

	return client.Database(dbName)
}
