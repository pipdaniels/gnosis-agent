package prospectivity

import (
	"context"
	"fmt"
	"math"
	"sort"

	"gnosis-agent/internal/config"
	"gnosis-agent/internal/models"
)

// ProspectivityAgent evaluates mineral prospectivity of targets
type ProspectivityAgent struct {
	config *config.ProspectivityAgentConfig
}

// NewProspectivityAgent creates a new prospectivity agent
func NewProspectivityAgent(cfg *config.ProspectivityAgentConfig) *ProspectivityAgent {
	return &ProspectivityAgent{config: cfg}
}

// Name returns the agent name
func (a *ProspectivityAgent) Name() string {
	return "prospectivity_agent"
}

// Initialize initializes the agent
func (a *ProspectivityAgent) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// Execute evaluates prospectivity based on anomaly results
func (a *ProspectivityAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	anomalyResults, ok := input.(*models.AnomalyResults)
	if !ok {
		return nil, fmt.Errorf("expected AnomalyResults, got %T", input)
	}

	// Identify targets from anomaly clusters and high-anomaly individual samples
	targets := a.identifyTargets(anomalyResults)

	// Score each target
	scoredTargets := make([]models.ProspectivityResult, 0, len(targets))
	for _, target := range targets {
		scored := a.scoreTarget(target, anomalyResults)
		scoredTargets = append(scoredTargets, scored)
	}

	// Sort by prospectivity score (highest first)
	sort.Slice(scoredTargets, func(i, j int) bool {
		return scoredTargets[i].ProspectivityScore > scoredTargets[j].ProspectivityScore
	})

	return &models.ProspectivityResults{
		OrgID:     anomalyResults.OrgID,
		DatasetID: anomalyResults.DatasetID,
		Targets:   scoredTargets,
	}, nil
}

// identifyTargets identifies potential targets from anomaly results
func (a *ProspectivityAgent) identifyTargets(anomalyResults *models.AnomalyResults) []Target {
	targets := make([]Target, 0)

	// Add cluster-based targets
	for _, cluster := range anomalyResults.Clusters {
		if len(cluster.SampleIDs) >= 2 { // At least 2 samples for a target
			targets = append(targets, Target{
				ID:          cluster.ClusterID,
				Location:    cluster.Centroid,
				SampleIDs:   cluster.SampleIDs,
				AvgAnomaly:  cluster.AvgAnomaly,
				IsCluster:   true,
			})
		}
	}

	// Add high-anomaly individual samples not in clusters
	clusterSamples := make(map[string]bool)
	for _, cluster := range anomalyResults.Clusters {
		for _, sampleID := range cluster.SampleIDs {
			clusterSamples[sampleID] = true
		}
	}

	targetID := len(targets)
	for _, result := range anomalyResults.Results {
		if result.AnomalyScore > 0.7 && !clusterSamples[result.SampleID] {
			// Get location from original data (would need to be passed through)
			// For now, use placeholder
			targets = append(targets, Target{
				ID:          fmt.Sprintf("T-%d", targetID),
				SampleIDs:   []string{result.SampleID},
				AvgAnomaly:  result.AnomalyScore,
				IsCluster:   false,
			})
			targetID++
		}
	}

	return targets
}

// scoreTarget calculates prospectivity score for a target
func (a *ProspectivityAgent) scoreTarget(target Target, anomalyResults *models.AnomalyResults) models.ProspectivityResult {
	// Scoring factors:
	// 1. Geochemical anomaly strength (60%)
	// 2. Number of anomalous samples (20%)
	// 3. Spatial continuity (20%)

	geochemScore := target.AvgAnomaly * a.config.Weights["geochemistry"]
	
	// Sample count score (normalized to max 1.0)
	sampleScore := math.Min(float64(len(target.SampleIDs))/10.0, 1.0) * a.config.Weights["sample_count"]
	
	// Continuity score (clusters score higher than isolated samples)
	continuityScore := 0.0
	if target.IsCluster {
		continuityScore = 1.0
	} else {
		continuityScore = 0.3
	}
	continuityScore *= a.config.Weights["continuity"]

	// Combined prospectivity score
	prospectivityScore := geochemScore + sampleScore + continuityScore

	// Normalize to [0, 1]
	if prospectivityScore > 1.0 {
		prospectivityScore = 1.0
	}

	// Calculate confidence
	confidence := a.calculateConfidence(target, anomalyResults)

	// Generate rationale
	rationale := a.generateRationale(target, geochemScore, sampleScore, continuityScore)

	// Contributing factors
	factors := map[string]float64{
		"geochemistry":       geochemScore,
		"sample_count":       sampleScore,
		"spatial_continuity": continuityScore,
	}

	return models.ProspectivityResult{
		TargetID:            target.ID,
		Location:            target.Location,
		ProspectivityScore:  prospectivityScore,
		Confidence:          confidence,
		Rationale:           rationale,
		ContributingFactors: factors,
	}
}

// calculateConfidence calculates confidence in the prospectivity score
func (a *ProspectivityAgent) calculateConfidence(target Target, anomalyResults *models.AnomalyResults) float64 {
	confidence := 0.5 // Base confidence

	// Higher confidence for clusters
	if target.IsCluster {
		confidence += 0.2
	}

	// Higher confidence for more samples
	if len(target.SampleIDs) >= 5 {
		confidence += 0.2
	} else if len(target.SampleIDs) >= 3 {
		confidence += 0.1
	}

	// Higher confidence for very strong anomalies
	if target.AvgAnomaly > 0.8 {
		confidence += 0.1
	}

	return math.Min(confidence, 1.0)
}

// generateRationale generates human-readable rationale
func (a *ProspectivityAgent) generateRationale(target Target, geochemScore, sampleScore, continuityScore float64) []string {
	rationale := make([]string, 0)

	// Anomaly strength
	if target.AvgAnomaly > 0.8 {
		rationale = append(rationale, fmt.Sprintf("Very strong geochemical anomaly (%.0f%% anomalous)", target.AvgAnomaly*100))
	} else if target.AvgAnomaly > 0.6 {
		rationale = append(rationale, fmt.Sprintf("Strong geochemical anomaly (%.0f%% anomalous)", target.AvgAnomaly*100))
	} else {
		rationale = append(rationale, fmt.Sprintf("Moderate geochemical anomaly (%.0f%% anomalous)", target.AvgAnomaly*100))
	}

	// Sample count
	if len(target.SampleIDs) >= 5 {
		rationale = append(rationale, fmt.Sprintf("Excellent sample coverage (%d samples)", len(target.SampleIDs)))
	} else if len(target.SampleIDs) >= 3 {
		rationale = append(rationale, fmt.Sprintf("Good sample coverage (%d samples)", len(target.SampleIDs)))
	} else {
		rationale = append(rationale, fmt.Sprintf("Limited sample coverage (%d sample)", len(target.SampleIDs)))
	}

	// Spatial continuity
	if target.IsCluster {
		rationale = append(rationale, "Spatially coherent anomaly cluster indicating potential mineralized zone")
	} else {
		rationale = append(rationale, "Isolated anomalous sample - may warrant follow-up sampling")
	}

	return rationale
}

// Shutdown cleans up resources
func (a *ProspectivityAgent) Shutdown(ctx context.Context) error {
	return nil
}

// Target represents a prospectivity target
type Target struct {
	ID         string
	Location   models.GeoLocation
	SampleIDs  []string
	AvgAnomaly float64
	IsCluster  bool
}
