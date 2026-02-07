package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/pipdaniels/geochem-agent/internal/config"
	"github.com/pipdaniels/geochem-agent/internal/models"
)

// OrchestratorAgent synthesizes all agent outputs into final decisions
type OrchestratorAgent struct {
	config *config.OrchestratorAgentConfig
}

// NewOrchestratorAgent creates a new orchestrator agent
func NewOrchestratorAgent(cfg *config.OrchestratorAgentConfig) *OrchestratorAgent {
	return &OrchestratorAgent{config: cfg}
}

// Name returns the agent name
func (a *OrchestratorAgent) Name() string {
	return "orchestrator_agent"
}

// Initialize initializes the agent
func (a *OrchestratorAgent) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// DecisionInput contains all inputs needed for decision making
type DecisionInput struct {
	QCResults            *models.QCResults
	AnomalyResults       *models.AnomalyResults
	ProspectivityResults *models.ProspectivityResults
	SamplingPlan         *models.SamplingPlan
}

// Execute synthesizes all inputs and makes drill decisions
func (a *OrchestratorAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	decisionInput, ok := input.(*DecisionInput)
	if !ok {
		return nil, fmt.Errorf("expected DecisionInput, got %T", input)
	}

	// Validate all required inputs
	if decisionInput.QCResults == nil ||
		decisionInput.AnomalyResults == nil ||
		decisionInput.ProspectivityResults == nil {
		return nil, fmt.Errorf("missing required agent outputs")
	}

	// Generate decisions for each target
	decisions := make([]models.DrillDecision, 0)

	for i, target := range decisionInput.ProspectivityResults.Targets {
		decision := a.makeDecision(target, decisionInput, i+1)
		decisions = append(decisions, decision)
	}

	return &models.DrillDecisions{
		OrgID:     decisionInput.ProspectivityResults.OrgID,
		DatasetID: decisionInput.ProspectivityResults.DatasetID,
		Decisions: decisions,
		CreatedAt: time.Now(),
	}, nil
}

// makeDecision makes a drill/no-drill decision for a target
func (a *OrchestratorAgent) makeDecision(target models.ProspectivityResult, input *DecisionInput, priority int) models.DrillDecision {
	// Decision logic:
	// 1. Must pass minimum confidence threshold
	// 2. Must have adequate QC
	// 3. Prospectivity and confidence determine recommendation

	decision := "needs_more_data"
	rationale := make([]string, 0)

	// Check QC threshold
	if input.QCResults.OverallQC < 0.7 {
		decision = "no_drill"
		rationale = append(rationale, fmt.Sprintf("Overall QC score too low: %.2f", input.QCResults.OverallQC))
	} else if target.Confidence < a.config.MinConfidence {
		decision = "needs_more_data"
		rationale = append(rationale, fmt.Sprintf("Confidence below threshold: %.2f < %.2f",
			target.Confidence, a.config.MinConfidence))
		rationale = append(rationale, "Recommend additional sampling before drilling decision")
	} else if target.ProspectivityScore >= 0.75 && target.Confidence >= 0.8 {
		decision = "drill"
		rationale = append(rationale, fmt.Sprintf("High prospectivity (%.0f%%) with high confidence (%.0f%%)",
			target.ProspectivityScore*100, target.Confidence*100))
		rationale = append(rationale, "Strong geochemical and spatial indicators support drilling")
	} else if target.ProspectivityScore >= 0.6 && target.Confidence >= a.config.MinConfidence {
		decision = "drill"
		rationale = append(rationale, fmt.Sprintf("Moderate prospectivity (%.0f%%) with adequate confidence (%.0f%%)",
			target.ProspectivityScore*100, target.Confidence*100))
		rationale = append(rationale, "Warrants exploratory drilling")
	} else if target.ProspectivityScore < 0.5 {
		decision = "no_drill"
		rationale = append(rationale, fmt.Sprintf("Low prospectivity: %.0f%%", target.ProspectivityScore*100))
		rationale = append(rationale, "Insufficient geochemical indicators")
	} else {
		decision = "needs_more_data"
		rationale = append(rationale, "Marginal prospectivity")
		rationale = append(rationale, "Additional sampling recommended to reduce uncertainty")
	}

	// Add contributing factor rationale
	rationale = append(rationale, target.Rationale...)

	// Calculate overall confidence
	overallConfidence := (target.Confidence * 0.6) + (input.QCResults.OverallQC * 0.4)

	// Aggregate agent inputs
	agentInputs := models.AgentInputSummary{
		QCScore:            input.QCResults.OverallQC,
		AnomalyScore:       a.getAverageAnomalyScore(target, input.AnomalyResults),
		ProspectivityScore: target.ProspectivityScore,
	}

	return models.DrillDecision{
		DecisionID:    fmt.Sprintf("D-%s-%d", input.ProspectivityResults.DatasetID, priority),
		TargetID:      target.TargetID,
		Decision:      decision,
		Confidence:    overallConfidence,
		Priority:      priority,
		Rationale:     rationale,
		AgentInputs:   agentInputs,
		HumanOverride: nil,
		CreatedAt:     time.Now(),
	}
}

// getAverageAnomalyScore gets average anomaly score for target samples
func (a *OrchestratorAgent) getAverageAnomalyScore(target models.ProspectivityResult, anomalyResults *models.AnomalyResults) float64 {
	// In practice, would match target to anomaly results
	// For now, use prospectivity score as proxy
	return target.ProspectivityScore * 0.85 // Slightly lower than prospectivity
}

// ApplyHumanOverride applies a human decision override
func (a *OrchestratorAgent) ApplyHumanOverride(decision *models.DrillDecision, userID, newDecision, justification string) {
	decision.HumanOverride = &models.HumanOverride{
		UserID:           userID,
		OriginalDecision: decision.Decision,
		NewDecision:      newDecision,
		Justification:    justification,
		Timestamp:        time.Now(),
	}

	decision.Decision = newDecision
}

// Shutdown cleans up resources
func (a *OrchestratorAgent) Shutdown(ctx context.Context) error {
	return nil
}
