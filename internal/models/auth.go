package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered user
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"` // Never return password hash in JSON
	APIKey       string             `bson:"api_key" json:"api_key"`
	Role         string             `bson:"role" json:"role"` // "admin", "user", "readonly"
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// Organization represents a tenant organization
// This is strictly for metadata; usually stored in a master database or configured statically
// In this multi-tenant setup, we might store this in the org's own DB for reference
type Organization struct {
	ID        string    `bson:"org_id" json:"org_id"`
	Name      string    `bson:"name" json:"name"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
