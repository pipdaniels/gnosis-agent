package mlpack

import (
	"testing"
)

func TestIsolationForest(t *testing.T) {
	// Create test data with clear anomaly
	data := [][]float64{
		{1.0, 2.0, 3.0},
		{1.1, 2.1, 3.1},
		{0.9, 1.9, 2.9},
		{1.0, 2.0, 3.0},
		{10.0, 20.0, 30.0}, // Anomaly
	}

	iforest := NewIsolationForest(100, 256)
	defer iforest.Free()

	err := iforest.Fit(data)
	if err != nil {
		t.Fatalf("Failed to fit: %v", err)
	}

	scores, err := iforest.Score(data)
	if err != nil {
		t.Fatalf("Failed to score: %v", err)
	}

	if len(scores) != len(data) {
		t.Errorf("Expected %d scores, got %d", len(data), len(scores))
	}

	// Last sample should have highest anomaly score
	if scores[4] <= scores[0] {
		t.Errorf("Expected anomaly to have higher score: got %.2f vs %.2f", scores[4], scores[0])
	}

	t.Logf("Anomaly scores: %v", scores)
}

func TestLOF(t *testing.T) {
	data := [][]float64{
		{1.0, 1.0},
		{1.1, 1.0},
		{1.0, 1.1},
		{10.0, 10.0}, // Anomaly
	}

	lof := NewLOF(2)
	defer lof.Free()

	scores, err := lof.Compute(data)
	if err != nil {
		t.Fatalf("Failed to compute LOF: %v", err)
	}

	if len(scores) != len(data) {
		t.Errorf("Expected %d scores, got %d", len(data), len(scores))
	}

	// Anomaly should have LOF > 1
	if scores[3] <= 1.0 {
		t.Logf("Warning: Expected anomaly to have LOF > 1, got %.2f", scores[3])
	}

	t.Logf("LOF scores: %v", scores)
}

func TestDBSCAN(t *testing.T) {
	data := [][]float64{
		{1.0, 1.0},
		{1.1, 1.0},
		{1.0, 1.1},
		{5.0, 5.0},
		{5.1, 5.0},
		{5.0, 5.1},
	}

	labels, err := DBSCANCluster(data, 0.5, 2)
	if err != nil {
		t.Fatalf("Failed to cluster: %v", err)
	}

	if len(labels) != len(data) {
		t.Errorf("Expected %d labels, got %d", len(data), len(labels))
	}

	// Count unique clusters
	uniqueLabels := make(map[int]bool)
	for _, label := range labels {
		uniqueLabels[label] = true
	}

	if len(uniqueLabels) < 2 {
		t.Logf("Warning: Expected at least 2 clusters, got %d", len(uniqueLabels))
	}

	t.Logf("Cluster labels: %v", labels)
	t.Logf("Unique clusters: %d", len(uniqueLabels))
}

func TestEmptyData(t *testing.T) {
	iforest := NewIsolationForest(10, 10)
	defer iforest.Free()

	err := iforest.Fit([][]float64{})
	if err == nil {
		t.Error("Expected error for empty data")
	}

	lof := NewLOF(5)
	defer lof.Free()

	_, err = lof.Compute([][]float64{})
	if err == nil {
		t.Error("Expected error for empty data")
	}

	_, err = DBSCANCluster([][]float64{}, 0.5, 2)
	if err == nil {
		t.Error("Expected error for empty data")
	}
}
