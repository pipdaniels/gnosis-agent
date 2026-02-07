package qc

import (
	"context"
	"fmt"
	"math"

	"github.com/pipdaniels/geochem-agent/internal/config"
	"github.com/pipdaniels/geochem-agent/internal/models"
	"gonum.org/v1/gonum/stat"
)

// QCAgent validates geochemical data quality
type QCAgent struct {
	config *config.QCAgentConfig
}

// NewQCAgent creates a new QC agent
func NewQCAgent(cfg *config.QCAgentConfig) *QCAgent {
	return &QCAgent{config: cfg}
}

// Name returns the agent name
func (a *QCAgent) Name() string {
	return "qc_agent"
}

// Initialize initializes the agent
func (a *QCAgent) Initialize(ctx context.Context, config interface{}) error {
	// Initialize any resources
	return nil
}

// Execute runs quality control on a processed dataset
func (a *QCAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	dataset, ok := input.(*models.ProcessedDataset)
	if !ok {
		return nil, fmt.Errorf("expected ProcessedDataset, got %T", input)
	}

	if len(dataset.ProcessedAssays) < a.config.MinSampleSize {
		return nil, fmt.Errorf("insufficient samples: got %d, need %d", len(dataset.ProcessedAssays), a.config.MinSampleSize)
	}

	results := make([]models.QCResult, 0, len(dataset.ProcessedAssays))

	for _, assay := range dataset.ProcessedAssays {
		qcResult := a.validateSample(assay, dataset)
		results = append(results, qcResult)
	}

	// Calculate overall QC score
	overallQC := a.calculateOverallQC(results)

	return &models.QCResults{
		OrgID:     dataset.OrgID,
		DatasetID: dataset.DatasetID,
		Results:   results,
		OverallQC: overallQC,
	}, nil
}

// validateSample performs QC checks on a single sample
func (a *QCAgent) validateSample(assay models.ProcessedAssay, dataset *models.ProcessedDataset) models.QCResult {
	flags := make([]models.QCFlag, 0)

	// Check 1: Statistical outliers using Z-scores
	for element, value := range assay.NormalizedElements {
		if math.Abs(value) > a.config.OutlierThreshold {
			flags = append(flags, models.QCFlag{
				Type:             "outlier",
				Severity:         "warning",
				Message:          fmt.Sprintf("%s value is %.2f standard deviations from mean", element, value),
				AffectedElements: []string{element},
			})
		}
	}

	// Check 2: Impossible values (negative concentrations)
	for element, value := range assay.RawElements {
		if value < 0 {
			flags = append(flags, models.QCFlag{
				Type:             "impossible_value",
				Severity:         "critical",
				Message:          fmt.Sprintf("%s has negative concentration: %.4f", element, value),
				AffectedElements: []string{element},
			})
		}
	}

	// Check 3: Extremely high values (potential contamination)
	for element, value := range assay.RawElements {
		if value > 1e6 { // More than 1000 ppm for trace elements
			flags = append(flags, models.QCFlag{
				Type:             "contamination",
				Severity:         "warning",
				Message:          fmt.Sprintf("%s has extremely high value: %.2f (possible contamination)", element, value),
				AffectedElements: []string{element},
			})
		}
	}

	// Check 4: Below detection limit handling
	belowDetectionCount := 0
	for _, isBelow := range assay.BelowDetectionFlags {
		if isBelow {
			belowDetectionCount++
		}
	}
	if belowDetectionCount > len(assay.BelowDetectionFlags)/2 {
		flags = append(flags, models.QCFlag{
			Type:     "data_quality",
			Severity: "warning",
			Message:  fmt.Sprintf("More than 50%% of elements below detection limit (%d/%d)", belowDetectionCount, len(assay.BelowDetectionFlags)),
		})
	}

	// Check 5: Missing critical data
	if len(assay.RawElements) == 0 {
		flags = append(flags, models.QCFlag{
			Type:     "missing_data",
			Severity: "critical",
			Message:  "No element data available",
		})
	}

	// Calculate QC score (1.0 = perfect, 0.0 = failed)
	// Each flag reduces score, critical flags reduce more
	qcScore := 1.0
	for _, flag := range flags {
		if flag.Severity == "critical" {
			qcScore -= 0.3
		} else if flag.Severity == "warning" {
			qcScore -= 0.1
		} else {
			qcScore -= 0.05
		}
	}

	if qcScore < 0 {
		qcScore = 0
	}

	passed := qcScore >= 0.7 && !hasCriticalFlags(flags)

	explanation := fmt.Sprintf("QC Score: %.2f (%d flags)", qcScore, len(flags))
	if !passed {
		if hasCriticalFlags(flags) {
			explanation += " - FAILED: Critical issues detected"
		} else {
			explanation += " - FAILED: Score below threshold"
		}
	}

	return models.QCResult{
		SampleID:    assay.SampleID,
		QCScore:     qcScore,
		Flags:       flags,
		Passed:      passed,
		Explanation: explanation,
	}
}

// calculateOverallQC calculates the overall QC score for the dataset
func (a *QCAgent) calculateOverallQC(results []models.QCResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, result := range results {
		sum += result.QCScore
	}
	return sum / float64(len(results))
}

// Shutdown cleans up agent resources
func (a *QCAgent) Shutdown(ctx context.Context) error {
	// No cleanup needed
	return nil
}

// hasCriticalFlags checks if any critical flags are present
func hasCriticalFlags(flags []models.QCFlag) bool {
	for _, flag := range flags {
		if flag.Severity == "critical" {
			return true
		}
	}
	return false
}

// CalculateElementStats calculates statistics for elements across the dataset
func (a *QCAgent) CalculateElementStats(dataset *models.ProcessedDataset) map[string]ElementStats {
	stats := make(map[string]ElementStats)

	// Collect all unique elements
	elements := make(map[string][]float64)
	for _, assay := range dataset.ProcessedAssays {
		for element, value := range assay.RawElements {
			elements[element] = append(elements[element], value)
		}
	}

	// Calculate stats for each element
	for element, values := range elements {
		if len(values) == 0 {
			continue
		}

		mean := stat.Mean(values, nil)
		variance := stat.Variance(values, nil)
		stddev := math.Sqrt(variance)

		stats[element] = ElementStats{
			Element: element,
			Count:   len(values),
			Mean:    mean,
			StdDev:  stddev,
			Min:     min(values),
			Max:     max(values),
		}
	}

	return stats
}

// ElementStats holds statistical information for an element
type ElementStats struct {
	Element string
	Count   int
	Mean    float64
	StdDev  float64
	Min     float64
	Max     float64
}

// Helper functions
func min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}
