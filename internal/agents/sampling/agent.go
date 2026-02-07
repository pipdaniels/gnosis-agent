package sampling

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/pipdaniels/geochem-agent/internal/config"
	"github.com/pipdaniels/geochem-agent/internal/models"
)

// SamplingAgent optimizes sampling locations for maximum information gain
type SamplingAgent struct {
	config *config.SamplingAgentConfig
}

// NewSamplingAgent creates a new sampling optimization agent
func NewSamplingAgent(cfg *config.SamplingAgentConfig) *SamplingAgent {
	return &SamplingAgent{config: cfg}
}

// Name returns the agent name
func (a *SamplingAgent) Name() string {
	return "sampling_agent"
}

// Initialize initializes the agent
func (a *SamplingAgent) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// Execute generates sampling recommendations based on prospectivity results
func (a *SamplingAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	prospectivityResults, ok := input.(*models.ProspectivityResults)
	if !ok {
		return nil, fmt.Errorf("expected ProspectivityResults, got %T", input)
	}

	budget := a.config.DefaultBudget

	// Generate sampling recommendations
	recommendations := a.generateRecommendations(prospectivityResults, budget)

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority // Lower priority number = higher priority
	})

	// Calculate total cost and expected ROI
	totalCost := 0.0
	for _, rec := range recommendations {
		totalCost += rec.EstimatedCost
	}

	expectedROI := a.calculateExpectedROI(recommendations, prospectivityResults)

	return &models.SamplingPlan{
		OrgID:           prospectivityResults.OrgID,
		DatasetID:       prospectivityResults.DatasetID,
		Budget:          budget,
		Recommendations: recommendations,
		TotalCost:       totalCost,
		ExpectedROI:     expectedROI,
	}, nil
}

// generateRecommendations creates sampling recommendations
func (a *SamplingAgent) generateRecommendations(prospectivityResults *models.ProspectivityResults, budget float64) []models.SamplingRecommendation {
	recommendations := make([]models.SamplingRecommendation, 0)
	priority := 1

	for _, target := range prospectivityResults.Targets {
		// High prospectivity targets get more detailed sampling
		samplesNeeded := a.calculateSamplesNeeded(target)

		for i := 0; i < samplesNeeded; i++ {
			// Generate location (in practice, use spatial algorithms)
			location := a.generateSampleLocation(target, i, samplesNeeded)

			expectedValue := a.calculateExpectedValue(target, i, samplesNeeded)

			rec := models.SamplingRecommendation{
				Location:      location,
				Priority:      priority,
				ExpectedValue: expectedValue,
				EstimatedCost: a.config.CostPerSample,
				Justification: fmt.Sprintf("Target %s (%.0f%% prospective): Sample %d/%d for optimal coverage",
					target.TargetID, target.ProspectivityScore*100, i+1, samplesNeeded),
			}

			recommendations = append(recommendations, rec)
			priority++

			// Stop if budget exceeded
			if float64(priority)*a.config.CostPerSample > budget {
				break
			}
		}

		if float64(priority)*a.config.CostPerSample > budget {
			break
		}

		if len(recommendations) >= a.config.MaxSuggestions {
			break
		}
	}

	return recommendations
}

// calculateSamplesNeeded determines how many samples are needed for a target
func (a *SamplingAgent) calculateSamplesNeeded(target models.ProspectivityResult) int {
	// More samples for higher prospectivity
	if target.ProspectivityScore > 0.8 {
		return 5 // Detailed sampling
	} else if target.ProspectivityScore > 0.6 {
		return 3 // Moderate sampling
	} else {
		return 2 // Reconnaissance sampling
	}
}

// generateSampleLocation generates a sample location around a target
func (a *SamplingAgent) generateSampleLocation(target models.ProspectivityResult, index, total int) models.GeoLocation {
	// Simple grid pattern around target center
	// In practice, use more sophisticated spatial optimization

	gridSpacing := a.config.GridResolution / 1000.0 // Convert meters to ~degrees

	// Circular pattern
	angle := (2.0 * math.Pi * float64(index)) / float64(total)
	radius := gridSpacing

	return models.GeoLocation{
		Latitude:  target.Location.Latitude + radius*math.Cos(angle),
		Longitude: target.Location.Longitude + radius*math.Sin(angle),
		Elevation: target.Location.Elevation,
	}
}

// calculateExpectedValue estimates information gain from a sample
func (a *SamplingAgent) calculateExpectedValue(target models.ProspectivityResult, sampleIndex, totalSamples int) float64 {
	// Expected value decreases with more samples (diminishing returns)
	baseValue := target.ProspectivityScore * target.Confidence

	// First samples have highest value
	positionFactor := 1.0 - (float64(sampleIndex) / float64(totalSamples) * 0.3)

	return baseValue * positionFactor
}

// calculateExpectedROI estimates return on investment
func (a *SamplingAgent) calculateExpectedROI(recommendations []models.SamplingRecommendation, prospectivityResults *models.ProspectivityResults) float64 {
	if len(recommendations) == 0 {
		return 0.0
	}

	totalValue := 0.0
	for _, rec := range recommendations {
		totalValue += rec.ExpectedValue
	}

	totalCost := float64(len(recommendations)) * a.config.CostPerSample

	if totalCost == 0 {
		return 0
	}

	// ROI = (value - cost) / cost
	// Normalize to 0-1 range for interpretability
	roi := totalValue / totalCost
	return math.Min(roi, 10.0) / 10.0 // Cap at 1.0
}

// Shutdown cleans up resources
func (a *SamplingAgent) Shutdown(ctx context.Context) error {
	return nil
}
