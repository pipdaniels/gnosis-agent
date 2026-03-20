package models

import "time"

// Organization represents a tenant organization
// This is strictly for metadata; usually stored in a master database or configured statically
// In this multi-tenant setup, we might store this in the org's own DB for reference
type Organization struct {
	ID        string    `bson:"org_id" json:"org_id"`
	Name      string    `bson:"name" json:"name"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
	CreatedBy User      `bson:"created_by" json:"created_by"`
	UpdatedBy User      `bson:"updated_by" json:"updated_by"`
	Members   []User    `bson:"members" json:"members"`
}
