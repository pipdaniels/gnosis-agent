package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pipdaniels/geochem-agent/internal/config"
	"github.com/pipdaniels/geochem-agent/internal/db"
	"github.com/pipdaniels/geochem-agent/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication logic
type AuthService struct {
	repo    db.AuthRepository
	mongo   *db.MongoManager
	config  *config.Config
}

// NewAuthService creates a new AuthService
func NewAuthService(repo db.AuthRepository, mongo *db.MongoManager, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:   repo,
		mongo:  mongo,
		config: cfg,
	}
}

// SignupRequest represents the data needed for signup
type SignupRequest struct {
	OrgName         string
	Email           string
	Password        string
	ConfirmPassword string
	Passkey         string
	TermsAccepted   bool
}

// LoginRequest represents the data needed for login
type LoginRequest struct {
	OrgID    string
	Email    string
	Password string
}

// SignupResult contains the result of a successful signup
type SignupResult struct {
	OrgID  string
	Token  string
	APIKey string
}

// Signup processes a new organization signup
func (s *AuthService) Signup(ctx context.Context, req SignupRequest) (*SignupResult, error) {
	// 1. Validation
	if req.Passkey != s.config.Security.SignupPasskey {
		return nil, errors.New("invalid passkey")
	}
	if !req.TermsAccepted {
		return nil, errors.New("terms must be accepted")
	}
	if req.Password != req.ConfirmPassword {
		return nil, errors.New("passwords do not match")
	}
	if len(req.Password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	// 2. Generate OrgID
	// Simple generation for now, could be more robust
	orgID := generateID(8)

	// 3. Create Org Database
	if err := s.mongo.CreateOrgDatabase(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to create org database: %w", err)
	}

	// 4. Connect to new DB
	db, err := s.mongo.GetOrgDatabase(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to org database: %w", err)
	}

	// 5. Check if user already exists (should be empty db, but safety check)
	existing, err := s.repo.GetUserByEmail(ctx, db, req.Email)
	if err == nil && existing != nil {
		return nil, errors.New("email already registered")
	}

	// 6. Create Organization Metadata
	org := &models.Organization{
		ID:   orgID,
		Name: req.OrgName,
	}
	if err := s.repo.CreateOrganization(ctx, db, org); err != nil {
		return nil, fmt.Errorf("failed to save organization: %w", err)
	}

	// 7. Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 8. Generate API Key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	// 9. Create User
	user := &models.User{
		ID:           primitive.NewObjectID(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		APIKey:       apiKey,
		Role:         "admin", // First user is admin
	}
	if err := s.repo.CreateUser(ctx, db, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 10. Generate JWT
	token, err := s.generateJWT(user, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &SignupResult{
		OrgID:  orgID,
		Token:  token,
		APIKey: apiKey,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (string, error) {
	// 1. Get Org DB
	db, err := s.mongo.GetOrgDatabase(req.OrgID)
	if err != nil {
		return "", errors.New("invalid organization ID")
	}

	// 2. Find User
	user, err := s.repo.GetUserByEmail(ctx, db, req.Email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", errors.New("invalid credentials")
		}
		return "", fmt.Errorf("database error: %w", err)
	}

	// 3. Verify Password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	// 4. Generate JWT
	return s.generateJWT(user, req.OrgID)
}

// ValidateAPIKey validates an API key for a specific organization
// Returns the user if valid
func (s *AuthService) ValidateAPIKey(ctx context.Context, apiKey string, orgID string) (*models.User, error) {
	db, err := s.mongo.GetOrgDatabase(orgID)
	if err != nil {
		return nil, errors.New("invalid organization ID")
	}

	user, err := s.repo.GetUserByAPIKey(ctx, db, apiKey)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid api key")
		}
		return nil, err
	}

	return user, nil
}

// Helper: Generate JWT
func (s *AuthService) generateJWT(user *models.User, orgID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.Hex(),
		"org_id":  orgID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 24 hour expiration
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.Security.JWTSecret))
}

// Helper: Generate random ID (alphanumeric)
func generateID(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:length]
}

// Helper: Generate API Key
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return "gk_" + base64.RawURLEncoding.EncodeToString(b), nil
}
