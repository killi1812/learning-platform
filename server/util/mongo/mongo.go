package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

var (
	ErrEmptyURI    = errors.New("mongodb uri is empty")
	ErrEmptyDbName = errors.New("mongodb database name is empty")
)

// NewMongoDb initializes and returns a MongoDB database connection.
func NewMongoDb(uri, dbName string) (*mongo.Database, error) {
	if uri == "" {
		zap.S().Errorf("MongoDB uri is empty")
		return nil, ErrEmptyURI
	}
	if dbName == "" {
		zap.S().Errorf("MongoDB database name is empty")
		return nil, ErrEmptyDbName
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		zap.S().Errorf("failed to create MongoDB client: %+v", err)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		zap.S().Warnf("MongoDB ping warning (server will retry on demand): %+v", err)
	} else {
		zap.S().Info("Successfully connected to MongoDB")
	}

	return client.Database(dbName), nil
}
