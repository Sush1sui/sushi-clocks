package repository

import (
	"context"
	"fmt"

	"github.com/sushi-clocks/backend/internal/domain"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
