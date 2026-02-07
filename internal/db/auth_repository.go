package db

import (
	"context"
	"time"

	"github.com/pipdaniels/geochem-agent/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AuthRepository handles authentication related database operations
type AuthRepository interface {
	CreateUser(ctx context.Context, db *mongo.Database, user *models.User) error
	GetUserByEmail(ctx context.Context, db *mongo.Database, email string) (*models.User, error)
	GetUserByAPIKey(ctx context.Context, db *mongo.Database, apiKey string) (*models.User, error)
	CreateOrganization(ctx context.Context, db *mongo.Database, org *models.Organization) error
}

// MongoAuthRepository implements AuthRepository
type MongoAuthRepository struct{}

// NewMongoAuthRepository creates a new MongoAuthRepository
func NewMongoAuthRepository() *MongoAuthRepository {
	return &MongoAuthRepository{}
}

// CreateUser creates a new user in the given database
func (r *MongoAuthRepository) CreateUser(ctx context.Context, db *mongo.Database, user *models.User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	
	_, err := db.Collection("users").InsertOne(ctx, user)
	return err
}

// GetUserByEmail retrieves a user by email
func (r *MongoAuthRepository) GetUserByEmail(ctx context.Context, db *mongo.Database, email string) (*models.User, error) {
	var user models.User
	err := db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByAPIKey retrieves a user by API key
func (r *MongoAuthRepository) GetUserByAPIKey(ctx context.Context, db *mongo.Database, apiKey string) (*models.User, error) {
	var user models.User
	err := db.Collection("users").FindOne(ctx, bson.M{"api_key": apiKey}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateOrganization creates a new organization record (metadata)
func (r *MongoAuthRepository) CreateOrganization(ctx context.Context, db *mongo.Database, org *models.Organization) error {
	org.CreatedAt = time.Now()
	_, err := db.Collection("organizations").InsertOne(ctx, org)
	return err
}
