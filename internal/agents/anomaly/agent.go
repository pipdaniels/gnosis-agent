package anomaly

import (
	"context"
	"fmt"
	"math"
	"sort"

	"gnosis-agent/internal/config"
	"gnosis-agent/internal/ml/mlpack"
	"gnosis-agent/internal/models"
)

// AnomalyAgent detects geochemical anomalies using ML algorithms
type AnomalyAgent struct {
	config *config.AnomalyAgentConfig
}

// NewAnomalyAgent creates a new anomaly detection agent
func NewAnomalyAgent(cfg *config.AnomalyAgentConfig) *AnomalyAgent {
	return &AnomalyAgent{config: cfg}
}

// Name returns the agent name
func (a *AnomalyAgent) Name() string {
	return "anomaly_agent"
}

// Initialize initializes the agent
func (a *AnomalyAgent) Initialize(ctx context.Context, config interface{}) error {
	return nil
}

// Execute runs anomaly detection on QC-passed dataset
func (a *AnomalyAgent) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	qcResults, ok := input.(*models.QCResults)
	if !ok {
		return nil, fmt.Errorf("expected QCResults, got %T", input)
	}

	// Filter to only QC-passed samples
	passedSampleIDs := make(map[string]bool)
	for _, result := range qcResults.Results {
		if result.Passed {
			passedSampleIDs[result.SampleID] = true
		}
	}

	if len(passedSampleIDs) == 0 {
		return nil, fmt.Errorf("no samples passed QC")
	}

	// Note: We need the original ProcessedDataset to get element values
	// In practice, this would be passed along with QCResults or retrieved from DB
	// For now, we'll return a placeholder and handle this in integration

	results := make([]models.AnomalyResult, 0)
	clusters := make([]models.AnomalyCluster, 0)

	return &models.AnomalyResults{
		OrgID:     qcResults.OrgID,
		DatasetID: qcResults.DatasetID,
		Results:   results,
		Clusters:  clusters,
	}, nil
}

// DetectAnomalies performs anomaly detection using multiple methods
func (a *AnomalyAgent) DetectAnomalies(dataset *models.ProcessedDataset, qcResults *models.QCResults) (*models.AnomalyResults, error) {
	// Filter to QC-passed samples
	passedSamples := make([]models.ProcessedAssay, 0)
	passedMap := make(map[string]bool)
	
	for _, qcResult := range qcResults.Results {
		if qcResult.Passed {
			passedMap[qcResult.SampleID] = true
		}
	}

	for _, assay := range dataset.ProcessedAssays {
		if passedMap[assay.SampleID] {
			passedSamples = append(passedSamples, assay)
		}
	}

	if len(passedSamples) == 0 {
		return nil, fmt.Errorf("no samples passed QC")
	}

	// Prepare data matrix for ML algorithms
	data, elements := a.prepareDataMatrix(passedSamples)

	// Method 1: Isolation Forest
	iforestScores := a.runIsolationForest(data)

	// Method 2: LOF
	lofScores := a.runLOF(data)

	// Combine scores
	combinedScores := a.combineScores(iforestScores, lofScores)

	// Calculate pathfinder scores
	pathfinderScores := a.calculatePathfinderScores(passedSamples, elements)

	// Create anomaly results
	results := make([]models.AnomalyResult, 0, len(passedSamples))
	for i, assay := range passedSamples {
		result := models.AnomalyResult{
			SampleID:         assay.SampleID,
			AnomalyScore:     combinedScores[i],
			PathfinderScores: pathfinderScores[assay.SampleID],
			Confidence:       a.calculateConfidence(combinedScores[i], iforestScores[i], lofScores[i]),
			RankedElements:   a.rankElements(pathfinderScores[assay.SampleID]),
		}
		results = append(results, result)
	}

	// Method 3: DBSCAN for spatial clustering of anomalies
	clusters := a.clusterAnomalies(passedSamples, combinedScores)

	// Assign cluster IDs to results
	for i := range results {
		for _, cluster := range clusters {
			for _, sampleID := range cluster.SampleIDs {
				if sampleID == results[i].SampleID {
					results[i].ClusterID = &cluster.ClusterID
					break
				}
			}
		}
	}

	return &models.AnomalyResults{
		OrgID:     dataset.OrgID,
		DatasetID: dataset.DatasetID,
		Results:   results,
		Clusters:  clusters,
	}, nil
}

// prepareDataMatrix converts assays to matrix format for ML
func (a *AnomalyAgent) prepareDataMatrix(assays []models.ProcessedAssay) ([][]float64, []string) {
	// Get all unique elements
	elementSet := make(map[string]bool)
	for _, assay := range assays {
		for element := range assay.NormalizedElements {
			elementSet[element] = true
		}
	}

	// Sort elements for consistent ordering
	elements := make([]string, 0, len(elementSet))
	for element := range elementSet {
		elements = append(elements, element)
	}
	sort.Strings(elements)

	// Create data matrix
	data := make([][]float64, len(assays))
	for i, assay := range assays {
		row := make([]float64, len(elements))
		for j, element := range elements {
			if val, exists := assay.NormalizedElements[element]; exists {
				row[j] = val
			} else {
				row[j] = 0.0
			}
		}
		data[i] = row
	}

	return data, elements
}

// runIsolationForest runs Isolation Forest anomaly detection
func (a *AnomalyAgent) runIsolationForest(data [][]float64) []float64 {
	iforest := mlpack.NewIsolationForest(
		a.config.IsolationForest.NumTrees,
		a.config.IsolationForest.SampleSize,
	)
	defer iforest.Free()

	if err := iforest.Fit(data); err != nil {
		// Return zeros on error
		return make([]float64, len(data))
	}

	scores, err := iforest.Score(data)
	if err != nil {
		return make([]float64, len(data))
	}

	// Normalize scores to [0, 1]
	return normalizeScores(scores)
}

// runLOF runs Local Outlier Factor
func (a *AnomalyAgent) runLOF(data [][]float64) []float64 {
	lof := mlpack.NewLOF(a.config.LOF.K)
	defer lof.Free()

	scores, err := lof.Compute(data)
	if err != nil {
		return make([]float64, len(data))
	}

	// LOF > 1 means outlier, convert to 0-1 scale
	normalized := make([]float64, len(scores))
	for i, score := range scores {
		if score > 1.0 {
			normalized[i] = math.Min((score-1.0)/2.0, 1.0)
		} else {
			normalized[i] = 0.0
		}
	}

	return normalized
}

// combineScores combines multiple anomaly scores
func (a *AnomalyAgent) combineScores(iforestScores, lofScores []float64) []float64 {
	combined := make([]float64, len(iforestScores))
	for i := range combined {
		// Weighted average
		combined[i] = (iforestScores[i]*0.6 + lofScores[i]*0.4)
	}
	return combined
}

// calculatePathfinderScores calculates pathfinder element scores
func (a *AnomalyAgent) calculatePathfinderScores(assays []models.ProcessedAssay, elements []string) map[string]map[string]float64 {
	scores := make(map[string]map[string]float64)

	for _, assay := range assays {
		sampleScores := make(map[string]float64)

		for element, value := range assay.NormalizedElements {
			weight := a.config.PathfinderWeights[element]
			if weight == 0 {
				weight = 0.1 // Default weight for non-pathfinder elements
			}

			// Higher normalized value * weight = higher pathfinder score
			elementScore := math.Abs(value) * weight
			sampleScores[element] = elementScore
		}

		scores[assay.SampleID] = sampleScores
	}

	return scores
}

// rankElements ranks elements by score
func (a *AnomalyAgent) rankElements(elementScores map[string]float64) []string {
	type elementScore struct {
		element string
		score   float64
	}

	scores := make([]elementScore, 0, len(elementScores))
	for element, score := range elementScores {
		scores = append(scores, elementScore{element, score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	ranked := make([]string, len(scores))
	for i, es := range scores {
		ranked[i] = es.element
	}

	return ranked
}

// calculateConfidence calculates confidence in anomaly score
func (a *AnomalyAgent) calculateConfidence(combined, iforest, lof float64) float64 {
	// Higher confidence when both methods agree
	agreement := 1.0 - math.Abs(iforest-lof)
	return (combined + agreement) / 2.0
}

// clusterAnomalies clusters anomalous samples spatially using DBSCAN
func (a *AnomalyAgent) clusterAnomalies(assays []models.ProcessedAssay, anomalyScores []float64) []models.AnomalyCluster {
	// Filter to high-anomaly samples (score > 0.6)
	anomalousIndices := make([]int, 0)
	for i, score := range anomalyScores {
		if score > 0.6 {
			anomalousIndices = append(anomalousIndices, i)
		}
	}

	if len(anomalousIndices) < 2 {
		return []models.AnomalyCluster{}
	}

	// Prepare spatial data (lat, lon)
	spatialData := make([][]float64, len(anomalousIndices))
	for i, idx := range anomalousIndices {
		spatialData[i] = []float64{
			assays[idx].Location.Latitude,
			assays[idx].Location.Longitude,
		}
	}

	// Run DBSCAN
	labels, err := mlpack.DBSCANCluster(spatialData, a.config.DBSCAN.Eps, a.config.DBSCAN.MinPts)
	if err != nil {
		return []models.AnomalyCluster{}
	}

	// Group by cluster
	clusterMap := make(map[int][]int) // clusterID -> sample indices
	for i, label := range labels {
		if label >= 0 { // -1 is noise in DBSCAN
			clusterMap[label] = append(clusterMap[label], anomalousIndices[i])
		}
	}

	// Create cluster objects
	clusters := make([]models.AnomalyCluster, 0, len(clusterMap))
	for clusterID, indices := range clusterMap {
		sampleIDs := make([]string, len(indices))
		var latSum, lonSum, scoreSum float64

		for i, idx := range indices {
			sampleIDs[i] = assays[idx].SampleID
			latSum += assays[idx].Location.Latitude
			lonSum += assays[idx].Location.Longitude
			scoreSum += anomalyScores[idx]
		}

		n := float64(len(indices))
		clusters = append(clusters, models.AnomalyCluster{
			ClusterID: fmt.Sprintf("C%d", clusterID),
			SampleIDs: sampleIDs,
			Centroid: models.GeoLocation{
				Latitude:  latSum / n,
				Longitude: lonSum / n,
			},
			AvgAnomaly: scoreSum / n,
		})
	}

	return clusters
}

// Shutdown cleans up resources
func (a *AnomalyAgent) Shutdown(ctx context.Context) error {
	return nil
}

// Helper functions

func normalizeScores(scores []float64) []float64 {
	if len(scores) == 0 {
		return scores
	}

	minScore := scores[0]
	maxScore := scores[0]

	for _, score := range scores {
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore == minScore {
		// All scores are the same
		normalized := make([]float64, len(scores))
		return normalized
	}

	normalized := make([]float64, len(scores))
	for i, score := range scores {
		normalized[i] = (score - minScore) / (maxScore - minScore)
	}

	return normalized
}
