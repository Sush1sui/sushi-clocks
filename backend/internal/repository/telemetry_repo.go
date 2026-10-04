package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/sushi-clocks/backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

// EnsureIndexes creates a 366-day fallback TTL index on timestamp.
func (r *TelemetryRepository) EnsureIndexes(ctx context.Context) error {
	if r == nil || r.collection == nil {
		return nil
	}
	model := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(366 * 24 * 3600),
	}
	_, err := r.collection.Indexes().CreateOne(ctx, model)
	if err != nil {
		return fmt.Errorf("create telemetry ttl index error: %w", err)
	}
	return nil
}

// GetRecordsForArchival finds unarchived telemetry records older than cutoff.
func (r *TelemetryRepository) GetRecordsForArchival(ctx context.Context, cutoff time.Time) ([]domain.PunchTelemetry, error) {
	if r == nil || r.collection == nil {
		return nil, nil
	}
	filter := bson.M{
		"timestamp": bson.M{"$lte": cutoff},
		"archived":  bson.M{"$ne": true},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find telemetry for archival error: %w", err)
	}
	defer cursor.Close(ctx)

	var list []domain.PunchTelemetry
	if err := cursor.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode telemetry for archival error: %w", err)
	}
	return list, nil
}

// DeleteArchivedRecords deletes telemetry records for a company older than cutoff.
func (r *TelemetryRepository) DeleteArchivedRecords(ctx context.Context, companyID string, cutoff time.Time) (int64, error) {
	if r == nil || r.collection == nil {
		return 0, nil
	}
	filter := bson.M{
		"company_id": companyID,
		"timestamp":  bson.M{"$lte": cutoff},
	}
	res, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("delete archived telemetry error: %w", err)
	}
	return res.DeletedCount, nil
}
