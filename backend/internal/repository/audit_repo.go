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

type AuditRepository struct {
	collection *mongo.Collection
}

func NewAuditRepository(db *mongo.Database) *AuditRepository {
	return &AuditRepository{
		collection: db.Collection("audit_logs"),
	}
}

func (r *AuditRepository) Insert(ctx context.Context, log domain.AuditLog) error {
	if r == nil || r.collection == nil {
		return nil
	}
	_, err := r.collection.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("insert audit log error: %w", err)
	}
	return nil
}

// EnsureIndexes creates a 366-day fallback TTL index on timestamp.
func (r *AuditRepository) EnsureIndexes(ctx context.Context) error {
	if r == nil || r.collection == nil {
		return nil
	}
	model := mongo.IndexModel{
		Keys:    bson.D{{Key: "timestamp", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(366 * 24 * 3600),
	}
	_, err := r.collection.Indexes().CreateOne(ctx, model)
	if err != nil {
		return fmt.Errorf("create audit log ttl index error: %w", err)
	}
	return nil
}

// GetRecordsForArchival finds unarchived audit log records older than cutoff.
func (r *AuditRepository) GetRecordsForArchival(ctx context.Context, cutoff time.Time) ([]domain.AuditLog, error) {
	if r == nil || r.collection == nil {
		return nil, nil
	}
	filter := bson.M{
		"timestamp": bson.M{"$lte": cutoff},
		"archived":  bson.M{"$ne": true},
	}
	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("find audit logs for archival error: %w", err)
	}
	defer cursor.Close(ctx)

	var list []domain.AuditLog
	if err := cursor.All(ctx, &list); err != nil {
		return nil, fmt.Errorf("decode audit logs for archival error: %w", err)
	}
	return list, nil
}

// DeleteArchivedRecords deletes audit records for a company older than cutoff.
func (r *AuditRepository) DeleteArchivedRecords(ctx context.Context, companyID string, cutoff time.Time) (int64, error) {
	if r == nil || r.collection == nil {
		return 0, nil
	}
	filter := bson.M{
		"company_id": companyID,
		"timestamp":  bson.M{"$lte": cutoff},
	}
	res, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("delete archived audit logs error: %w", err)
	}
	return res.DeletedCount, nil
}
