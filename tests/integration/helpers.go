package integration

import (
	"fmt"
	"time"

	"github.com/pipdaniels/geochem-agent/internal/models"
)

// CreateTestDataset creates a synthetic geochemical dataset for testing
func CreateTestDataset() *models.ProcessedDataset {
	return &models.ProcessedDataset{
		OrgID:     "testOrgID",
		DatasetID: "testDatasetID",
		ProcessedAssays: []models.ProcessedAssay{
			// Normal samples
			{
				SampleID: "S001",
				Location: models.GeoLocation{
					Latitude:  -30.5,
					Longitude: 145.2,
					Elevation: 500,
				},
				NormalizedElements: map[string]float64{
					"Au": 0.2,
					"Cu": 0.3,
					"Mo": 0.1,
				},
				RawElements: map[string]float64{
					"Au": 0.05,
					"Cu": 150.0,
					"Mo": 20.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
					"Mo": false,
				},
			},
			{
				SampleID: "S002",
				Location: models.GeoLocation{
					Latitude:  -30.51,
					Longitude: 145.21,
					Elevation: 505,
				},
				NormalizedElements: map[string]float64{
					"Au": 0.3,
					"Cu": 0.4,
					"Mo": 0.2,
				},
				RawElements: map[string]float64{
					"Au": 0.06,
					"Cu": 160.0,
					"Mo": 25.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
					"Mo": false,
				},
			},
			// High anomaly samples (cluster)
			{
				SampleID: "S003",
				Location: models.GeoLocation{
					Latitude:  -30.55,
					Longitude: 145.25,
					Elevation: 520,
				},
				NormalizedElements: map[string]float64{
					"Au": 5.5, // High anomaly
					"Cu": 4.2,
					"Mo": 3.8,
				},
				RawElements: map[string]float64{
					"Au": 2.5,
					"Cu": 850.0,
					"Mo": 180.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
					"Mo": false,
				},
			},
			{
				SampleID: "S004",
				Location: models.GeoLocation{
					Latitude:  -30.56,
					Longitude: 145.26,
					Elevation: 525,
				},
				NormalizedElements: map[string]float64{
					"Au": 6.2, // High anomaly
					"Cu": 5.1,
					"Mo": 4.5,
				},
				RawElements: map[string]float64{
					"Au": 3.1,
					"Cu": 920.0,
					"Mo": 200.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
					"Mo": false,
				},
			},
			{
				SampleID: "S005",
				Location: models.GeoLocation{
					Latitude:  -30.57,
					Longitude: 145.27,
					Elevation: 530,
				},
				NormalizedElements: map[string]float64{
					"Au": 5.8,
					"Cu": 4.9,
					"Mo": 4.2,
				},
				RawElements: map[string]float64{
					"Au": 2.8,
					"Cu": 890.0,
					"Mo": 195.0,
				},
				BelowDetectionFlags: map[string]bool{
					"Au": false,
					"Cu": false,
					"Mo": false,
				},
			},
		},
		Transformations: []string{"z-score", "clr"},
		ProcessedAt:     time.Now(),
	}
}

// CreateLargeTestDataset creates a larger synthetic dataset
func CreateLargeTestDataset(numSamples int) *models.ProcessedDataset {
	dataset := CreateTestDataset()
	
	// Add more samples
	for i := len(dataset.ProcessedAssays); i < numSamples; i++ {
		sample := models.ProcessedAssay{
			SampleID: generateSampleID(i),
			Location: generateLocation(i),
			NormalizedElements: map[string]float64{
				"Au": randomNormal(0.5, 0.2),
				"Cu": randomNormal(0.5, 0.2),
				"Mo": randomNormal(0.3, 0.15),
			},
			RawElements: map[string]float64{
				"Au": randomNormal(0.1, 0.05),
				"Cu": randomNormal(200, 50),
				"Mo": randomNormal(30, 10),
			},
			BelowDetectionFlags: map[string]bool{
				"Au": false,
				"Cu": false,
				"Mo": false,
			},
		}
		dataset.ProcessedAssays = append(dataset.ProcessedAssays, sample)
	}
	
	return dataset
}

func generateSampleID(index int) string {
	return fmt.Sprintf("S%03d", index+1)
}

func generateLocation(index int) models.GeoLocation {
	return models.GeoLocation{
		Latitude:  -30.5 + float64(index)*0.01,
		Longitude: 145.2 + float64(index)*0.01,
		Elevation: 500 + float64(index)*5,
	}
}

// Simple random normal distribution simulator (Box-Muller transform)
func randomNormal(mean, stddev float64) float64 {
	// For testing, use deterministic values
	return mean + stddev*0.5
}
