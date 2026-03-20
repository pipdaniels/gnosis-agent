package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a registered or invited user within an organization
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"` // Never return password hash in JSON
	FirstName    string             `bson:"first_name" json:"first_name"`
	LastName     string             `bson:"last_name" json:"last_name"`
	APIKey       string             `bson:"api_key" json:"api_key"`
	Role         string             `bson:"role" json:"role"`             // "admin", "analyst", "viewer"
	Status       string             `bson:"status" json:"status"`         // "joined", "pending"
	IsActive     bool               `bson:"is_active" json:"is_active"`
	InviteToken  string             `bson:"invite_token,omitempty" json:"-"`
	DeletedAt    *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}
