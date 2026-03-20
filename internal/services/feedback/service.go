package feedback

import (
	"context"
	"fmt"

	"gnosis-agent/internal/db"
	"gnosis-agent/internal/models"
	"gnosis-agent/internal/services/orgcontext"

	"go.mongodb.org/mongo-driver/bson"
)

// FeedbackService processes drilling outcomes to improve future decisions
type FeedbackService struct {
	db         *db.MongoManager
	orgContext *orgcontext.OrgContextService
}

// FeedbackAnalysis contains analysis of drilling outcomes
type FeedbackAnalysis struct {
	OrgID              string
	TotalDecisions     int
	TotalOutcomes      int
	SuccessRate        float64
	FalsePositives     int // Predicted drill, no mineralization
	FalseNegatives     int // Predicted no drill, found mineralization
	TruePositives      int // Predicted drill, found mineralization
	TrueNegatives      int // Predicted no drill, no mineralization
	AverageConfidence  float64
	TopPathfinders     map[string]float64
	RecommendedChanges map[string]interface{}
}

// NewFeedbackService creates a new feedback service
func NewFeedbackService(dbManager *db.MongoManager, orgCtx *orgcontext.OrgContextService) *FeedbackService {
	return &FeedbackService{
		db:         dbManager,
		orgContext: orgCtx,
	}
}

// RecordDrillOutcome stores drilling results linked to a decision
func (s *FeedbackService) RecordDrillOutcome(ctx context.Context, decisionID string, outcome models.DrillOutcome) error {
	// First, find the decision log
	// We need to determine which org this decision belongs to
	// This requires searching across org databases or storing orgID in decisionID
	
	// For now, we'll extract orgID from context or decision structure
	// In practice, you'd pass orgID as a parameter
	
	return fmt.Errorf("not implemented: need orgID parameter")
}

// RecordDrillOutcomeWithOrg stores drilling results with explicit org ID
func (s *FeedbackService) RecordDrillOutcomeWithOrg(ctx context.Context, orgID, decisionID string, outcome models.DrillOutcome) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("decision_logs")
	
	// Update the decision log with the outcome
	update := bson.M{
		"$set": bson.M{
			"outcome": outcome,
		},
	}
	
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"decision_id": decisionID},
		update,
	)
	
	if err != nil {
		return fmt.Errorf("failed to record drill outcome: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("decision not found: %s", decisionID)
	}
	
	return nil
}

// AnalyzeFeedback analyzes all drilling outcomes for an organization
func (s *FeedbackService) AnalyzeFeedback(ctx context.Context, orgID string) (*FeedbackAnalysis, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	// Find all decision logs with outcomes
	cursor, err := collection.Find(ctx, bson.M{
		"outcome": bson.M{"$exists": true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query decision logs: %w", err)
	}
	defer cursor.Close(ctx)
	
	analysis := &FeedbackAnalysis{
		OrgID:              orgID,
		TopPathfinders:     make(map[string]float64),
		RecommendedChanges: make(map[string]interface{}),
	}
	
	var totalConfidence float64
	// pathfinderSuccesses := make(map[string]int)
	// pathfinderTotal := make(map[string]int)
	
	for cursor.Next(ctx) {
		var log models.DecisionLog
		if err := cursor.Decode(&log); err != nil {
			continue
		}
		
		analysis.TotalOutcomes++
		
		decision := log.Decision.Decision
		success := log.Outcome.Success
		
		totalConfidence += log.Decision.Confidence
		
		// Classify outcome
		if decision == "drill" && success {
			analysis.TruePositives++
		} else if decision == "drill" && !success {
			analysis.FalsePositives++
		} else if decision == "no_drill" && success {
			analysis.FalseNegatives++
		} else if decision == "no_drill" && !success {
			analysis.TrueNegatives++
		}
		
		// Track pathfinder performance (would need element data from original analysis)
		// This is simplified - in practice, correlate with anomaly results
	}
	
	if analysis.TotalOutcomes > 0 {
		analysis.SuccessRate = float64(analysis.TruePositives+analysis.TrueNegatives) / float64(analysis.TotalOutcomes)
		analysis.AverageConfidence = totalConfidence / float64(analysis.TotalOutcomes)
	}
	
	// Generate recommendations
	s.generateRecommendations(analysis)
	
	return analysis, nil
}

// generateRecommendations creates recommendations based on feedback analysis
func (s *FeedbackService) generateRecommendations(analysis *FeedbackAnalysis) {
	// If too many false positives, increase confidence threshold
	if analysis.FalsePositives > analysis.TruePositives {
		analysis.RecommendedChanges["increase_confidence_threshold"] = true
		analysis.RecommendedChanges["suggested_threshold"] = 0.8
	}
	
	// If too many false negatives, decrease confidence threshold
	if analysis.FalseNegatives > analysis.TrueNegatives {
		analysis.RecommendedChanges["decrease_confidence_threshold"] = true
		analysis.RecommendedChanges["suggested_threshold"] = 0.6
	}
	
	// If success rate is high, can be more aggressive
	if analysis.SuccessRate > 0.8 {
		analysis.RecommendedChanges["risk_tolerance"] = "aggressive"
	} else if analysis.SuccessRate < 0.5 {
		analysis.RecommendedChanges["risk_tolerance"] = "conservative"
	}
}

// UpdateModelWeights adjusts agent weights based on success/failure patterns
func (s *FeedbackService) UpdateModelWeights(ctx context.Context, orgID string) error {
	analysis, err := s.AnalyzeFeedback(ctx, orgID)
	if err != nil {
		return err
	}
	
	// Apply recommended threshold changes
	if threshold, ok := analysis.RecommendedChanges["suggested_threshold"].(float64); ok {
		thresholds := map[string]float64{
			"min_confidence": threshold,
		}
		
		if err := s.orgContext.UpdateThresholds(ctx, orgID, thresholds); err != nil {
			return fmt.Errorf("failed to update thresholds: %w", err)
		}
	}
	
	// Apply risk tolerance changes
	if tolerance, ok := analysis.RecommendedChanges["risk_tolerance"].(string); ok {
		if err := s.orgContext.SetRiskTolerance(ctx, orgID, tolerance); err != nil {
			return fmt.Errorf("failed to update risk tolerance: %w", err)
		}
	}
	
	return nil
}

// GetSuccessRate returns the current success rate for an organization
func (s *FeedbackService) GetSuccessRate(ctx context.Context, orgID string) (float64, error) {
	analysis, err := s.AnalyzeFeedback(ctx, orgID)
	if err != nil {
		return 0, err
	}
	
	return analysis.SuccessRate, nil
}
