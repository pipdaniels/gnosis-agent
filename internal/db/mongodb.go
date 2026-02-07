package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoManager manages MongoDB connections and database-level multi-tenancy
type MongoManager struct {
	client         *mongo.Client
	databasePrefix string
	timeout        time.Duration
}

// NewMongoManager creates a new MongoDB manager with database-level multi-tenancy
func NewMongoManager(uri, databasePrefix string, timeout int) (*MongoManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &MongoManager{
		client:         client,
		databasePrefix: databasePrefix,
		timeout:        time.Duration(timeout) * time.Second,
	}, nil
}

// GetOrgDatabase returns the MongoDB database for a specific organization
// Database naming: {prefix}_org_{orgID}
func (m *MongoManager) GetOrgDatabase(orgID string) (*mongo.Database, error) {
	if orgID == "" {
		return nil, fmt.Errorf("orgID cannot be empty")
	}

	dbName := fmt.Sprintf("%s_org_%s", m.databasePrefix, orgID)
	return m.client.Database(dbName), nil
}

// CreateOrgDatabase creates and initializes a new database for an organization
func (m *MongoManager) CreateOrgDatabase(ctx context.Context, orgID string) error {
	db, err := m.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}

	// Create collections and indexes
	collections := []string{
		"raw_datasets",
		"processed_datasets",
		"qc_results",
		"anomaly_results",
		"prospectivity_results",
		"sampling_recommendations",
		"decisions",
		"decision_logs",
		"org_context",
		"reports",
		"users",
		"organizations",
	}

	for _, collName := range collections {
		// Create collection if it doesn't exist
		err := db.CreateCollection(ctx, collName)
		if err != nil {
			// Ignore error if collection already exists
			if !isDuplicateKeyError(err) {
				return fmt.Errorf("failed to create collection %s: %w", collName, err)
			}
		}
	}

	// Create indexes
	if err := m.createIndexes(ctx, db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates necessary indexes for all collections
func (m *MongoManager) createIndexes(ctx context.Context, db *mongo.Database) error {
	// Index for raw_datasets
	_, err := db.Collection("raw_datasets").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"dataset_id": 1},
	})
	if err != nil {
		return err
	}

	// Index for processed_datasets
	_, err = db.Collection("processed_datasets").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"dataset_id": 1},
	})
	if err != nil {
		return err
	}

	// Index for decisions
	_, err = db.Collection("decisions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"decision_id": 1},
	})
	if err != nil {
		return err
	}

	// Index for decision_logs with timestamp
	_, err = db.Collection("decision_logs").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"timestamp": -1},
	})
	if err != nil {
		return err
	}

	// Index for reports
	_, err = db.Collection("reports").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"report_id": 1},
	})
	if err != nil {
		return err
	}

	// Index for users (unique email)
	_, err = db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]interface{}{"email": 1},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Index for users (api key)
	_, err = db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"api_key": 1},
	})
	if err != nil {
		return err
	}

	return nil
}

// ListOrgDatabases lists all organization databases
func (m *MongoManager) ListOrgDatabases(ctx context.Context) ([]string, error) {
	databases, err := m.client.ListDatabaseNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	var orgDatabases []string
	prefix := fmt.Sprintf("%s_org_", m.databasePrefix)
	for _, dbName := range databases {
		if len(dbName) > len(prefix) && dbName[:len(prefix)] == prefix {
			orgDatabases = append(orgDatabases, dbName)
		}
	}

	return orgDatabases, nil
}

// DropOrgDatabase drops an organization's database (admin only - use with caution!)
func (m *MongoManager) DropOrgDatabase(ctx context.Context, orgID string) error {
	db, err := m.GetOrgDatabase(orgID)
	if err != nil {
		return err
	}

	return db.Drop(ctx)
}

// Close closes the MongoDB connection
func (m *MongoManager) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// GetClient returns the underlying MongoDB client
func (m *MongoManager) GetClient() *mongo.Client {
	return m.client
}

// Helper function to check if error is duplicate key error
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	// MongoDB error code 48 is NamespaceExists
	return mongo.IsDuplicateKeyError(err) ||
		(err != nil && err.Error() == "collection already exists")
}
