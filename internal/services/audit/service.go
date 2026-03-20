package audit

import (
	"context"
	"fmt"
	"time"

	"gnosis-agent/internal/db"
	"gnosis-agent/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AuditService provides complete audit trail for decisions
type AuditService struct {
	db *db.MongoManager
}

// NewAuditService creates a new audit service
func NewAuditService(dbManager *db.MongoManager) *AuditService {
	return &AuditService{
		db: dbManager,
	}
}

// LogDecision stores complete decision context for audit trail
func (s *AuditService) LogDecision(ctx context.Context, orgID string, log models.DecisionLog) error {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}
	
	collection := database.Collection("decision_logs")
	
	// Set creation time if not set
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	
	_, err = collection.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("failed to log decision: %w", err)
	}
	
	return nil
}

// GetDecisionHistory retrieves decision history for a dataset
func (s *AuditService) GetDecisionHistory(ctx context.Context, orgID, datasetID string) ([]models.DecisionLog, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	filter := bson.M{"dataset_id": datasetID}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query decision history: %w", err)
	}
	defer cursor.Close(ctx)
	
	var logs []models.DecisionLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode decision logs: %w", err)
	}
	
	return logs, nil
}

// GetAgentTraces retrieves agent execution traces for a specific decision
func (s *AuditService) GetAgentTraces(ctx context.Context, orgID, decisionID string) ([]models.AgentTrace, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	var log models.DecisionLog
	err = collection.FindOne(ctx, bson.M{"decision_id": decisionID}).Decode(&log)
	if err != nil {
		return nil, fmt.Errorf("decision not found: %w", err)
	}
	
	return log.AgentTraces, nil
}

// CompareDecisions finds similar past decisions for comparison
func (s *AuditService) CompareDecisions(ctx context.Context, orgID string, currentDecision models.DrillDecision) ([]models.DecisionLog, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	// Find similar decisions based on:
	// - Similar prospectivity scores
	// - Similar anomaly scores
	// - Same decision type
	
	filter := bson.M{
		"decision.decision": currentDecision.Decision,
		"decision.agent_inputs.prospectivity_score": bson.M{
			"$gte": currentDecision.AgentInputs.ProspectivityScore - 0.2,
			"$lte": currentDecision.AgentInputs.ProspectivityScore + 0.2,
		},
	}
	
	opts := options.Find().SetLimit(10).SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find similar decisions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var logs []models.DecisionLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode decision logs: %w", err)
	}
	
	return logs, nil
}

// GetDecisionByID retrieves a specific decision log
func (s *AuditService) GetDecisionByID(ctx context.Context, orgID, decisionID string) (*models.DecisionLog, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	var log models.DecisionLog
	err = collection.FindOne(ctx, bson.M{"decision_id": decisionID}).Decode(&log)
	if err != nil {
		return nil, fmt.Errorf("decision not found: %w", err)
	}
	
	return &log, nil
}

// GetRecentDecisions retrieves the most recent decisions for an organization
func (s *AuditService) GetRecentDecisions(ctx context.Context, orgID string, limit int) ([]models.DecisionLog, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	opts := options.Find().SetLimit(int64(limit)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent decisions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var logs []models.DecisionLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("failed to decode decision logs: %w", err)
	}
	
	return logs, nil
}

// GetDecisionStats returns statistics about decisions
func (s *AuditService) GetDecisionStats(ctx context.Context, orgID string) (map[string]interface{}, error) {
	database, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	collection := database.Collection("decision_logs")
	
	// Count by decision type
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$decision.decision",
				"count": bson.M{"$sum": 1},
			},
		},
	}
	
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate stats: %w", err)
	}
	defer cursor.Close(ctx)
	
	stats := make(map[string]interface{})
	decisionCounts := make(map[string]int)
	
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		decisionCounts[result.ID] = result.Count
	}
	
	stats["by_decision_type"] = decisionCounts
	stats["total_decisions"] = sumCounts(decisionCounts)
	
	return stats, nil
}

func sumCounts(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}
