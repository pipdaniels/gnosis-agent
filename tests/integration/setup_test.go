package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"gnosis-agent/internal/config"
	"gnosis-agent/internal/db"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	testDB        *db.MongoManager
	testOrgID     = "TEST_ORG"
	testDatasetID = "TEST_DS_001"
)

func TestMain(m *testing.M) {
	// Setup test environment
	setup()
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	cleanup()
	
	os.Exit(code)
}

func setup() {
	// Use test MongoDB instance
	mongoURI := os.Getenv("TEST_MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	
	var err error
	testDB, err = db.NewMongoManager(mongoURI, "geoagent_test", 10)
	if err != nil {
		panic("Failed to connect to test MongoDB: " + err.Error())
	}
	
	// Create test org database
	ctx := context.Background()
	if err := testDB.CreateOrgDatabase(ctx, testOrgID); err != nil {
		panic("Failed to create test org database: " + err.Error())
	}
}

func cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Drop test org database
	if err := testDB.DropOrgDatabase(ctx, testOrgID); err != nil {
		// Log but don't fail
		println("Warning: Failed to cleanup test database:", err.Error())
	}
	
	// Close connection
	if err := testDB.Close(ctx); err != nil {
		println("Warning: Failed to close MongoDB connection:", err.Error())
	}
}

func getTestDB(t *testing.T) *mongo.Database {
	db, err := testDB.GetOrgDatabase(testOrgID)
	if err != nil {
		t.Fatalf("Failed to get test database: %v", err)
	}
	return db
}

func getTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		MongoDB: config.MongoConfig{
			URI:      "mongodb://localhost:27017",
			Database: "geoagent_test",
			Timeout:  10,
		},
		Org: config.OrgConfig{
			ID:            testOrgID,
			Name:          "Test Organization",
			DepositModels: []string{"porphyry", "epithermal"},
		},
		Agents: config.AgentConfig{
			QC: config.QCAgentConfig{
				OutlierThreshold:    3.0,
				MinSampleSize:       3,
				EnableLabComparison: true,
			},
			Anomaly: config.AnomalyAgentConfig{
				PathfinderWeights: map[string]float64{
					"Au": 0.4,
					"Cu": 0.3,
					"Mo": 0.2,
				},
				IsolationForest: config.IsolationForestConfig{
					NumTrees:   100,
					SampleSize: 256,
				},
				LOF: config.LOFConfig{
					K: 20,
				},
				DBSCAN: config.DBSCANConfig{
					Eps:    0.5,
					MinPts: 5,
				},
			},
			Prospectivity: config.ProspectivityAgentConfig{
				GridResolution: 100.0,
				Weights: map[string]float64{
					"geochemistry":   0.6,
					"sample_count":   0.2,
					"continuity":     0.2,
				},
			},
			Sampling: config.SamplingAgentConfig{
				DefaultBudget:  50000.0,
				CostPerSample:  500.0,
				MaxSuggestions: 20,
			},
			Orchestrator: config.OrchestratorAgentConfig{
				MinConfidence:      0.7,
				RequireHumanReview: false,
			},
		},
	}
}
