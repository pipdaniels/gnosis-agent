package qc

import (
	"math"
	"time"

	"gnosis-agent/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const zScoreThreshold = 3.0

// RunQC performs quality control checks on a raw dataset.
// It runs three checks:
//  1. Z-Score outliers per element (severity: warning / critical)
//  2. Missing / below-detection values (element value == 0)
//  3. Contamination proxy: sample with >3 element concentrations 3× MAD above median
func RunQC(orgID string, dataset models.RawDataset) models.QCResults {
	assays := dataset.RawAssays

	// Collect all element keys
	elementKeys := make(map[string]bool)
	for _, a := range assays {
		for k := range a.Elements {
			elementKeys[k] = true
		}
	}

	// Compute mean & stddev per element for Z-score
	means := make(map[string]float64)
	stds := make(map[string]float64)
	for elem := range elementKeys {
		var sum, sumSq float64
		var n float64
		for _, a := range assays {
			if v, ok := a.Elements[elem]; ok && v > 0 {
				sum += v
				sumSq += v * v
				n++
			}
		}
		if n > 1 {
			mean := sum / n
			variance := (sumSq / n) - (mean * mean)
			if variance < 0 {
				variance = 0
			}
			means[elem] = mean
			stds[elem] = math.Sqrt(variance)
		}
	}

	// Compute median per element for contamination check
	medians := make(map[string]float64)
	mads := make(map[string]float64)
	for elem := range elementKeys {
		var vals []float64
		for _, a := range assays {
			if v, ok := a.Elements[elem]; ok && v > 0 {
				vals = append(vals, v)
			}
		}
		if len(vals) == 0 {
			continue
		}
		// Simple O(n^2) median for small datasets
		med := median(vals)
		medians[elem] = med
		var absDevs []float64
		for _, v := range vals {
			absDevs = append(absDevs, math.Abs(v-med))
		}
		mads[elem] = median(absDevs)
	}

	var results []models.QCResult
	passCount := 0

	for _, assay := range assays {
		var flags []models.QCFlag
		passed := true

		// --- Check 1: Outlier (Z-score) ---
		for elem, val := range assay.Elements {
			if stds[elem] == 0 {
				continue
			}
			z := math.Abs(val-means[elem]) / stds[elem]
			if z > zScoreThreshold {
				severity := "warning"
				if z > 5 {
					severity = "critical"
				}
				flags = append(flags, models.QCFlag{
					Type:             "outlier",
					Severity:         severity,
					Message:          "Z-score exceeds threshold",
					AffectedElements: []string{elem},
				})
				passed = false
			}
		}

		// --- Check 2: Missing / below-detection ---
		var missingElems []string
		for elem := range elementKeys {
			v, exists := assay.Elements[elem]
			if !exists || v == 0 {
				missingElems = append(missingElems, elem)
			}
		}
		if len(missingElems) > 0 {
			flags = append(flags, models.QCFlag{
				Type:             "missing_data",
				Severity:         "warning",
				Message:          "Missing or below-detection element values",
				AffectedElements: missingElems,
			})
			passed = false
		}

		// --- Check 3: Contamination (>3 elements at 3x MAD) ---
		contamCount := 0
		var contamElems []string
		for elem, val := range assay.Elements {
			if mads[elem] > 0 && math.Abs(val-medians[elem]) > 3*mads[elem] {
				contamCount++
				contamElems = append(contamElems, elem)
			}
		}
		if contamCount > 3 {
			flags = append(flags, models.QCFlag{
				Type:             "contamination",
				Severity:         "critical",
				Message:          "Multiple elements exceed 3× MAD — possible contamination",
				AffectedElements: contamElems,
			})
			passed = false
		}

		score := 1.0
		if !passed {
			// Score degrades with number of flags
			score = math.Max(0, 1.0-float64(len(flags))*0.2)
		} else {
			passCount++
		}

		results = append(results, models.QCResult{
			SampleID:    assay.SampleID,
			QCScore:     score,
			Flags:       flags,
			Passed:      passed,
			Explanation: buildExplanation(flags),
		})
	}

	overallQC := 0.0
	if len(assays) > 0 {
		overallQC = float64(passCount) / float64(len(assays))
	}

	return models.QCResults{
		ID:        primitive.NewObjectID(),
		OrgID:     orgID,
		DatasetID: dataset.DatasetID,
		Results:   results,
		OverallQC: overallQC,
		CreatedAt: time.Now(),
	}
}

func buildExplanation(flags []models.QCFlag) string {
	if len(flags) == 0 {
		return "All checks passed."
	}
	msg := ""
	for i, f := range flags {
		if i > 0 {
			msg += "; "
		}
		msg += f.Message
	}
	return msg
}

// median returns the median of a float64 slice (does not sort in-place).
func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	// Bubble sort (small slices expected)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}
