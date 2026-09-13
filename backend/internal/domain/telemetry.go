package domain

import "time"

type PunchTelemetry struct {
	ID        string    `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID string    `bson:"company_id" json:"company_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Action    string    `bson:"action" json:"action"` // "clock_in" or "clock_out"
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	IP        string    `bson:"ip" json:"ip"`
	Device    string    `bson:"device" json:"device"`
	OS        string    `bson:"os" json:"os"`
	Browser   string    `bson:"browser" json:"browser"`
}
