package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GeoLocation represents a geographic location
type GeoLocation struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
	Elevation float64 `bson:"elevation,omitempty" json:"elevation,omitempty"` // meters
}

// Assay represents a single geochemical assay sample
type Assay struct {
	SampleID    string             `bson:"sample_id" json:"sample_id"`
	Location    GeoLocation        `bson:"location" json:"location"`
	Elements    map[string]float64 `bson:"elements" json:"elements"` // element symbol -> concentration
	LabName     string             `bson:"lab_name" json:"lab_name"`
	CollectedAt time.Time          `bson:"collected_at" json:"collected_at"`
	Comments    string             `bson:"comments,omitempty" json:"comments,omitempty"`
	Deleted     bool               `bson:"deleted,omitempty" json:"deleted,omitempty"`
}

// DatasetMetadata holds metadata about a dataset
type DatasetMetadata struct {
	SamplingMethod    string             `bson:"sampling_method" json:"sampling_method"`       // e.g., "soil", "rock chip", "drill core"
	DetectionLimits   map[string]float64 `bson:"detection_limits" json:"detection_limits"`     // element -> detection limit
	ProjectName       string             `bson:"project_name" json:"project_name"`
	GeologicalSetting string             `bson:"geological_setting" json:"geological_setting"`
	DepositType       string             `bson:"deposit_type,omitempty" json:"deposit_type,omitempty"`
	UploadedBy        string             `bson:"uploaded_by" json:"uploaded_by"`
	Notes             string             `bson:"notes,omitempty" json:"notes,omitempty"`
}

// RawDataset represents a raw geochemical dataset
type RawDataset struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID      string             `bson:"org_id" json:"org_id"`
	DatasetID  string             `bson:"dataset_id" json:"dataset_id"`
	RawAssays  []Assay            `bson:"raw_assays" json:"raw_assays"`
	Metadata   DatasetMetadata    `bson:"metadata" json:"metadata"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
}

// ProcessedAssay represents a normalized and transformed assay
type ProcessedAssay struct {
	SampleID           string             `bson:"sample_id" json:"sample_id"`
	Location           GeoLocation        `bson:"location" json:"location"`
	NormalizedElements map[string]float64 `bson:"normalized_elements" json:"normalized_elements"` // Z-score normalized
	LogRatioElements   map[string]float64 `bson:"log_ratio_elements" json:"log_ratio_elements"`   // CLR transformed
	RawElements        map[string]float64 `bson:"raw_elements" json:"raw_elements"`               // Original values
	BelowDetectionFlags map[string]bool   `bson:"below_detection_flags" json:"below_detection_flags"`
}

// ProcessedDataset represents a processed geochemical dataset
type ProcessedDataset struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID            string             `bson:"org_id" json:"org_id"`
	DatasetID        string             `bson:"dataset_id" json:"dataset_id"`
	ProcessedAssays  []ProcessedAssay   `bson:"processed_assays" json:"processed_assays"`
	Transformations  []string           `bson:"transformations" json:"transformations"` // List of applied transformations
	ProcessedAt      time.Time          `bson:"processed_at" json:"processed_at"`
}

// QCFlag represents a quality control flag
type QCFlag struct {
	Type             string   `bson:"type" json:"type"`                           // "outlier", "contamination", "lab_bias", "missing_data"
	Severity         string   `bson:"severity" json:"severity"`                   // "critical", "warning", "info"
	Message          string   `bson:"message" json:"message"`
	AffectedElements []string `bson:"affected_elements" json:"affected_elements"`
}

// QCResult represents quality control results for a sample
type QCResult struct {
	SampleID    string   `bson:"sample_id" json:"sample_id"`
	QCScore     float64  `bson:"qc_score" json:"qc_score"`         // 0.0 - 1.0
	Flags       []QCFlag `bson:"flags" json:"flags"`
	Passed      bool     `bson:"passed" json:"passed"`
	Explanation string   `bson:"explanation" json:"explanation"`
}

// QCResults represents QC results for an entire dataset
type QCResults struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID      string             `bson:"org_id" json:"org_id"`
	DatasetID  string             `bson:"dataset_id" json:"dataset_id"`
	Results    []QCResult         `bson:"results" json:"results"`
	OverallQC  float64            `bson:"overall_qc" json:"overall_qc"` // Average QC score
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

// AnomalyResult represents anomaly detection results for a sample
type AnomalyResult struct {
	SampleID         string             `bson:"sample_id" json:"sample_id"`
	AnomalyScore     float64            `bson:"anomaly_score" json:"anomaly_score"`         // 0.0 - 1.0
	PathfinderScores map[string]float64 `bson:"pathfinder_scores" json:"pathfinder_scores"` // element -> score
	ClusterID        *string            `bson:"cluster_id,omitempty" json:"cluster_id,omitempty"`
	Confidence       float64            `bson:"confidence" json:"confidence"`
	RankedElements   []string           `bson:"ranked_elements" json:"ranked_elements"` // Most anomalous first
}

// AnomalyResults represents anomaly detection results for a dataset
type AnomalyResults struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID     string             `bson:"org_id" json:"org_id"`
	DatasetID string             `bson:"dataset_id" json:"dataset_id"`
	Results   []AnomalyResult    `bson:"results" json:"results"`
	Clusters  []AnomalyCluster   `bson:"clusters" json:"clusters"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// AnomalyCluster represents a spatial cluster of anomalous samples
type AnomalyCluster struct {
	ClusterID   string        `bson:"cluster_id" json:"cluster_id"`
	SampleIDs   []string      `bson:"sample_ids" json:"sample_ids"`
	Centroid    GeoLocation   `bson:"centroid" json:"centroid"`
	AvgAnomaly  float64       `bson:"avg_anomaly" json:"avg_anomaly"`
}

// ProspectivityResult represents a prospectivity target
type ProspectivityResult struct {
	TargetID            string             `bson:"target_id" json:"target_id"`
	Location            GeoLocation        `bson:"location" json:"location"`
	ProspectivityScore  float64            `bson:"prospectivity_score" json:"prospectivity_score"` // 0.0 - 1.0
	Confidence          float64            `bson:"confidence" json:"confidence"`
	Rationale           []string           `bson:"rationale" json:"rationale"`
	ContributingFactors map[string]float64 `bson:"contributing_factors" json:"contributing_factors"`
}

// ProspectivityResults represents prospectivity analysis for a dataset
type ProspectivityResults struct {
	ID        primitive.ObjectID    `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID     string                `bson:"org_id" json:"org_id"`
	DatasetID string                `bson:"dataset_id" json:"dataset_id"`
	Targets   []ProspectivityResult `bson:"targets" json:"targets"`
	CreatedAt time.Time             `bson:"created_at" json:"created_at"`
}

// SamplingRecommendation represents a recommended sample location
type SamplingRecommendation struct {
	Location      GeoLocation `bson:"location" json:"location"`
	Priority      int         `bson:"priority" json:"priority"`
	ExpectedValue float64     `bson:"expected_value" json:"expected_value"`
	EstimatedCost float64     `bson:"estimated_cost" json:"estimated_cost"`
	Justification string      `bson:"justification" json:"justification"`
}

// SamplingPlan represents a sampling optimization plan
type SamplingPlan struct {
	ID              primitive.ObjectID       `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID           string                   `bson:"org_id" json:"org_id"`
	DatasetID       string                   `bson:"dataset_id" json:"dataset_id"`
	Budget          float64                  `bson:"budget" json:"budget"`
	Recommendations []SamplingRecommendation `bson:"recommendations" json:"recommendations"`
	TotalCost       float64                  `bson:"total_cost" json:"total_cost"`
	ExpectedROI     float64                  `bson:"expected_roi" json:"expected_roi"`
	CreatedAt       time.Time                `bson:"created_at" json:"created_at"`
}

// HumanOverride represents a human override of an agent decision
type HumanOverride struct {
	UserID           string    `bson:"user_id" json:"user_id"`
	OriginalDecision string    `bson:"original_decision" json:"original_decision"`
	NewDecision      string    `bson:"new_decision" json:"new_decision"`
	Justification    string    `bson:"justification" json:"justification"`
	Timestamp        time.Time `bson:"timestamp" json:"timestamp"`
}

// AgentInputSummary summarizes inputs from all agents
type AgentInputSummary struct {
	QCScore            float64 `bson:"qc_score" json:"qc_score"`
	AnomalyScore       float64 `bson:"anomaly_score" json:"anomaly_score"`
	ProspectivityScore float64 `bson:"prospectivity_score" json:"prospectivity_score"`
}

// DrillDecision represents a drill/no-drill decision
type DrillDecision struct {
	DecisionID    string             `bson:"decision_id" json:"decision_id"`
	TargetID      string             `bson:"target_id" json:"target_id"`
	Decision      string             `bson:"decision" json:"decision"` // "drill", "no_drill", "needs_more_data"
	Confidence    float64            `bson:"confidence" json:"confidence"`
	Priority      int                `bson:"priority" json:"priority"`
	Rationale     []string           `bson:"rationale" json:"rationale"`
	AgentInputs   AgentInputSummary  `bson:"agent_inputs" json:"agent_inputs"`
	HumanOverride *HumanOverride     `bson:"human_override,omitempty" json:"human_override,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
}

// DrillDecisions represents all decisions for a dataset
type DrillDecisions struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID     string             `bson:"org_id" json:"org_id"`
	DatasetID string             `bson:"dataset_id" json:"dataset_id"`
	Decisions []DrillDecision    `bson:"decisions" json:"decisions"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// Intercept represents a drill intercept
type Intercept struct {
	From     float64            `bson:"from" json:"from"` // meters
	To       float64            `bson:"to" json:"to"`     // meters
	Grade    map[string]float64 `bson:"grade" json:"grade"`
	Minerals []string           `bson:"minerals" json:"minerals"`
}

// DrillOutcome represents the outcome of a drilling decision
type DrillOutcome struct {
	Success       bool               `bson:"success" json:"success"`
	Intercepts    []Intercept        `bson:"intercepts" json:"intercepts"`
	Grade         map[string]float64 `bson:"grade" json:"grade"`
	FeedbackNotes string             `bson:"feedback_notes" json:"feedback_notes"`
	RecordedAt    time.Time          `bson:"recorded_at" json:"recorded_at"`
}

// AgentTrace represents execution trace of an agent
type AgentTrace struct {
	AgentName     string      `bson:"agent_name" json:"agent_name"`
	ExecutionTime int64       `bson:"execution_time_ms" json:"execution_time_ms"`
	Input         interface{} `bson:"input" json:"input"`
	Output        interface{} `bson:"output" json:"output"`
	Reasoning     []string    `bson:"reasoning" json:"reasoning"`
}

// DecisionLog represents a complete audit log of a decision
type DecisionLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID       string             `bson:"org_id" json:"org_id"`
	DecisionID  string             `bson:"decision_id" json:"decision_id"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Decision    DrillDecision      `bson:"decision" json:"decision"`
	InputData   interface{}        `bson:"input_data" json:"input_data"`
	AgentTraces []AgentTrace       `bson:"agent_traces" json:"agent_traces"`
	Outcome     *DrillOutcome      `bson:"outcome,omitempty" json:"outcome,omitempty"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	
}

// MineralOccurrence represents a known mineral occurrence
type MineralOccurrence struct {
	Location    GeoLocation        `bson:"location" json:"location"`
	DepositType string             `bson:"deposit_type" json:"deposit_type"`
	Commodities []string           `bson:"commodities" json:"commodities"`
	Grade       map[string]float64 `bson:"grade" json:"grade"`
}

// ModelArtifact represents a trained ML model
type ModelArtifact struct {
	ModelType   string    `bson:"model_type" json:"model_type"` // "isolation_forest", "lof", etc.
	Version     string    `bson:"version" json:"version"`
	TrainedAt   time.Time `bson:"trained_at" json:"trained_at"`
	Accuracy    float64   `bson:"accuracy" json:"accuracy"`
	DataPath    string    `bson:"data_path" json:"data_path"` // Path to serialized model
}

// OrgContext represents organization-specific context and learning
type OrgContext struct {
	ID                  primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	OrgID               string              `bson:"org_id" json:"org_id"`
	DepositModels       []string            `bson:"deposit_models" json:"deposit_models"`
	KnownMineralization []MineralOccurrence `bson:"known_mineralization" json:"known_mineralization"`
	RiskTolerance       string              `bson:"risk_tolerance" json:"risk_tolerance"` // "conservative", "balanced", "aggressive"
	CustomThresholds    map[string]float64  `bson:"custom_thresholds" json:"custom_thresholds"`
	ModelArtifacts      []ModelArtifact     `bson:"model_artifacts" json:"model_artifacts"`
	UpdatedAt           time.Time           `bson:"updated_at" json:"updated_at"`
}
