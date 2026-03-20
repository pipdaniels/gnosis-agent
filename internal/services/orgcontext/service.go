package orgcontext

import (
	"context"
	"fmt"
	"time"

	"gnosis-agent/internal/db"
	"gnosis-agent/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// OrgContextService manages organization-specific context and learning
type OrgContextService struct {
	db *db.MongoManager
}

// NewOrgContextService creates a new org context service
func NewOrgContextService(dbManager *db.MongoManager) *OrgContextService {
	return &OrgContextService{
		db: dbManager,
	}
}

// GetContext retrieves organization-specific context
func (s *OrgContextService) GetContext(ctx context.Context, orgID string) (*models.OrgContext, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}

	collection := database.Collection("org_context")
	
	var orgContext models.OrgContext
	err = collection.FindOne(ctx, bson.M{"org_id": orgID}).Decode(&orgContext)
	
	if err == mongo.ErrNoDocuments {
		// Create default context
		return s.createDefaultContext(ctx, orgID)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get org context: %w", err)
	}
	
	return &orgContext, nil
}

// createDefaultContext creates a default context for a new org
func (s *OrgContextService) createDefaultContext(ctx context.Context, orgID string) (*models.OrgContext, error) {
	defaultContext := &models.OrgContext{
		OrgID:               orgID,
		DepositModels:       []string{"porphyry"},
		KnownMineralization: []models.MineralOccurrence{},
		RiskTolerance:       "balanced",
		CustomThresholds:    make(map[string]float64),
		ModelArtifacts:      []models.ModelArtifact{},
		UpdatedAt:           time.Now(),
	}
	
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("org_context")
	_, err = collection.InsertOne(ctx, defaultContext)
	if err != nil {
		return nil, fmt.Errorf("failed to create default context: %w", err)
	}
	
	return defaultContext, nil
}

// UpdateThresholds updates custom thresholds for an organization
func (s *OrgContextService) UpdateThresholds(ctx context.Context, orgID string, thresholds map[string]float64) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$set": bson.M{
			"custom_thresholds": thresholds,
			"updated_at":        time.Now(),
		},
	}
	
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update thresholds: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("org context not found for org %s", orgID)
	}
	
	return nil
}

// RecordMineralization stores a new mineral discovery
func (s *OrgContextService) RecordMineralization(ctx context.Context, orgID string, occurrence models.MineralOccurrence) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$push": bson.M{
			"known_mineralization": occurrence,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	if err != nil {
		return fmt.Errorf("failed to record mineralization: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("org context not found for org %s", orgID)
	}
	
	return nil
}

// UpdatePathfinderWeights updates pathfinder element weights
func (s *OrgContextService) UpdatePathfinderWeights(ctx context.Context, orgID string, weights map[string]float64) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$set": bson.M{
			"custom_thresholds.pathfinder_weights": weights,
			"updated_at":                           time.Now(),
		},
	}
	
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	return err
}

// GetDepositModels returns configured deposit models
func (s *OrgContextService) GetDepositModels(ctx context.Context, orgID string) ([]string, error) {
	orgContext, err := s.GetContext(ctx, orgID)
	if err != nil {
		return nil, err
	}
	
	return orgContext.DepositModels, nil
}

// UpdateDepositModels updates the deposit models for an organization
func (s *OrgContextService) UpdateDepositModels(ctx context.Context, orgID string, models []string) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$set": bson.M{
			"deposit_models": models,
			"updated_at":     time.Now(),
		},
	}
	
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	return err
}

// SetRiskTolerance sets the organization's risk tolerance level
func (s *OrgContextService) SetRiskTolerance(ctx context.Context, orgID string, tolerance string) error {
	validTolerances := map[string]bool{
		"conservative": true,
		"balanced":     true,
		"aggressive":   true,
	}
	
	if !validTolerances[tolerance] {
		return fmt.Errorf("invalid risk tolerance: %s (must be conservative, balanced, or aggressive)", tolerance)
	}
	
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$set": bson.M{
			"risk_tolerance": tolerance,
			"updated_at":     time.Now(),
		},
	}
	
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	return err
}

// StoreModelArtifact stores a trained model artifact
func (s *OrgContextService) StoreModelArtifact(ctx context.Context, orgID string, artifact models.ModelArtifact) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("org_context")
	
	update := bson.M{
		"$push": bson.M{
			"model_artifacts": artifact,
		},
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"org_id": orgID},
		update,
	)
	
	return err
}
