package domain

import "time"

type AuditLog struct {
	ID         string    `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID  string    `bson:"company_id" json:"company_id"`
	Collection string    `bson:"collection" json:"collection"`
	Action     string    `bson:"action" json:"action"`
	ActorID    string    `bson:"actor_id" json:"actor_id"`
	TargetID   string    `bson:"target_id" json:"target_id"`
	Before     any       `bson:"before,omitempty" json:"before,omitempty"`
	After      any       `bson:"after,omitempty" json:"after,omitempty"`
	Reason     string    `bson:"reason,omitempty" json:"reason,omitempty"`
	Timestamp  time.Time `bson:"timestamp" json:"timestamp"`
	Archived   bool      `bson:"archived,omitempty" json:"archived,omitempty"`
}
