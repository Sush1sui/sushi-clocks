package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// NewMongoClient establishes and verifies a connection to MongoDB Atlas
func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongo uri is empty")
	}

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect error: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping error: %w", err)
	}

	return client, nil
}
