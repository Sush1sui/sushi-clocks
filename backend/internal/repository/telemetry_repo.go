package repository

import (
	"context"
	"fmt"

	"github.com/sushi-clocks/backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type TelemetryRepository struct {
	collection *mongo.Collection
}

func NewTelemetryRepository(db *mongo.Database) *TelemetryRepository {
	return &TelemetryRepository{
		collection: db.Collection("punch_telemetry"),
	}
}

func (r *TelemetryRepository) Insert(ctx context.Context, t domain.PunchTelemetry) error {
	if r == nil || r.collection == nil {
		return nil
	}
	_, err := r.collection.InsertOne(ctx, t)
	if err != nil {
		return fmt.Errorf("insert punch telemetry error: %w", err)
	}
	return nil
}
