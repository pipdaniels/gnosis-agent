package qc

import (
	"context"
	"testing"
	"time"

	"github.com/pipdaniels/geochem-agent/internal/config"
	"github.com/pipdaniels/geochem-agent/internal/models"
)

func TestQCAgent_Execute(t *testing.T) {
	cfg := &config.QCAgentConfig{
		OutlierThreshold:    3.0,
		MinSampleSize:       3,
		EnableLabComparison: true,
	}

	agent := NewQCAgent(cfg)

	// Create test dataset
	dataset := &models.ProcessedDataset{
		OrgID:     "TEST_ORG",
		DatasetID: "TEST_DS",
		ProcessedAssays: []models.ProcessedAssay{
			{
				SampleID: "S1",
				NormalizedElements: map[string]float64{
					"Au": 0.5,  // Normal
					"Cu": 1.2,  // Normal
				},
				RawElements: map[string]float64{
					"Au": 0.01,
					"Cu": 100.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
				},
			},
			{
				SampleID: "S2",
				NormalizedElements: map[string]float64{
					"Au": 4.5,  // Outlier!
					"Cu": 1.0,
				},
				RawElements: map[string]float64{
					"Au": 10.0,
					"Cu": 95.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
				},
			},
			{
				SampleID: "S3",
				NormalizedElements: map[string]float64{
					"Au": -0.3,
					"Cu": 0.8,
				},
				RawElements: map[string]float64{
					"Au": -0.5, // Impossible value!
					"Cu": 80.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
				},
			},
		},
		ProcessedAt: time.Now(),
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, dataset)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	qcResults, ok := result.(*models.QCResults)
	if !ok {
		t.Fatalf("Expected QCResults, got %T", result)
	}

	// Verify results
	if len(qcResults.Results) != 3 {
		t.Errorf("Expected 3 QC results, got %d", len(qcResults.Results))
	}

	// Check S1 (should pass)
	s1 := qcResults.Results[0]
	if !s1.Passed {
		t.Errorf("S1 should pass QC")
	}
	if len(s1.Flags) > 0 {
		t.Logf("S1 flags: %+v", s1.Flags)
	}

	// Check S2 (should have outlier warning)
	s2 := qcResults.Results[1]
	hasOutlierFlag := false
	for _, flag := range s2.Flags {
		if flag.Type == "outlier" {
			hasOutlierFlag = true
		}
	}
	if !hasOutlierFlag {
		t.Errorf("S2 should have outlier flag")
	}

	// Check S3 (should fail with critical flag)
	s3 := qcResults.Results[2]
	if s3.Passed {
		t.Errorf("S3 should fail QC (has impossible value)")
	}
	hasCritical := false
	for _, flag := range s3.Flags {
		if flag.Severity == "critical" {
			hasCritical = true
		}
	}
	if !hasCritical {
		t.Errorf("S3 should have critical flag")
	}

	t.Logf("Overall QC Score: %.2f", qcResults.OverallQC)
}

func TestQCAgent_BelowDetectionLimit(t *testing.T) {
	cfg := &config.QCAgentConfig{
		OutlierThreshold: 3.0,
		MinSampleSize:    1,
	}

	agent := NewQCAgent(cfg)

	dataset := &models.ProcessedDataset{
		OrgID:     "TEST_ORG",
		DatasetID: "TEST_DS",
		ProcessedAssays: []models.ProcessedAssay{
			{
				SampleID: "S1",
				NormalizedElements: map[string]float64{
					"Au": 0.0,
					"Cu": 0.0,
					"Ag": 0.0,
				},
				RawElements: map[string]float64{
					"Au": 0.001,
					"Cu": 0.001,
					"Ag": 0.001,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": true,  // Below detection
					"Cu": true,  // Below detection
					"Ag": false,
				},
			},
		},
	}

	ctx := context.Background()
	result, err := agent.Execute(ctx, dataset)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	qcResults := result.(*models.QCResults)
	s1 := qcResults.Results[0]

	// Should have warning about high below-detection count
	hasWarning := false
	for _, flag := range s1.Flags {
		if flag.Type == "data_quality" {
			hasWarning = true
			t.Logf("Flag: %s", flag.Message)
		}
	}

	if !hasWarning {
		t.Errorf("Expected warning about below detection limit")
	}
}

func TestQCAgent_InsufficientSamples(t *testing.T) {
	cfg := &config.QCAgentConfig{
		OutlierThreshold: 3.0,
		MinSampleSize:    10,
	}

	agent := NewQCAgent(cfg)

	dataset := &models.ProcessedDataset{
		OrgID:     "TEST_ORG",
		DatasetID: "TEST_DS",
		ProcessedAssays: []models.ProcessedAssay{
			{SampleID: "S1"},
		},
	}

	ctx := context.Background()
	_, err := agent.Execute(ctx, dataset)
	if err == nil {
		t.Error("Expected error for insufficient samples")
	}
}

func TestCalculateElementStats(t *testing.T) {
	cfg := &config.QCAgentConfig{}
	agent := NewQCAgent(cfg)

	dataset := &models.ProcessedDataset{
		ProcessedAssays: []models.ProcessedAssay{
			{
				RawElements: map[string]float64{
					"Au": 1.0,
					"Cu": 100.0,
				},
			},
			{
				RawElements: map[string]float64{
					"Au": 2.0,
					"Cu": 200.0,
				},
			},
			{
				RawElements: map[string]float64{
					"Au": 3.0,
					"Cu": 300.0,
				},
			},
		},
	}

	stats := agent.CalculateElementStats(dataset)

	auStats, exists := stats["Au"]
	if !exists {
		t.Error("Expected Au stats")
	}

	if auStats.Count != 3 {
		t.Errorf("Expected 3 Au samples, got %d", auStats.Count)
	}

	if auStats.Mean != 2.0 {
		t.Errorf("Expected Au mean of 2.0, got %.2f", auStats.Mean)
	}

	if auStats.Min != 1.0 || auStats.Max != 3.0 {
		t.Errorf("Expected Au range [1.0, 3.0], got [%.2f, %.2f]", auStats.Min, auStats.Max)
	}

	t.Logf("Au stats: %+v", auStats)
	t.Logf("Cu stats: %+v", stats["Cu"])
}
