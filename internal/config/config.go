package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	MongoDB  MongoConfig    `mapstructure:"mongodb"`
	Org      OrgConfig      `mapstructure:"org"`
	Agents   AgentConfig    `mapstructure:"agents"`
	Security SecurityConfig `mapstructure:"security"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"` // Prefix for org databases
	Timeout  int    `mapstructure:"timeout"`
}

// OrgConfig holds organization-specific configuration
type OrgConfig struct {
	ID            string   `mapstructure:"id"`
	Name          string   `mapstructure:"name"`
	DepositModels []string `mapstructure:"deposit_models"`
}

// AgentConfig holds configuration for all agents
type AgentConfig struct {
	QC             QCAgentConfig             `mapstructure:"qc"`
	Anomaly        AnomalyAgentConfig        `mapstructure:"anomaly"`
	Prospectivity  ProspectivityAgentConfig  `mapstructure:"prospectivity"`
	Sampling       SamplingAgentConfig       `mapstructure:"sampling"`
	Orchestrator   OrchestratorAgentConfig   `mapstructure:"orchestrator"`
}

// QCAgentConfig holds QC agent specific configuration
type QCAgentConfig struct {
	OutlierThreshold    float64 `mapstructure:"outlier_threshold"`
	MinSampleSize       int     `mapstructure:"min_sample_size"`
	EnableLabComparison bool    `mapstructure:"enable_lab_comparison"`
}

// AnomalyAgentConfig holds anomaly detection agent configuration
type AnomalyAgentConfig struct {
	PathfinderWeights map[string]float64 `mapstructure:"pathfinder_weights"`
	IsolationForest   IsolationForestConfig `mapstructure:"isolation_forest"`
	LOF               LOFConfig             `mapstructure:"lof"`
	DBSCAN            DBSCANConfig          `mapstructure:"dbscan"`
}

// IsolationForestConfig holds Isolation Forest parameters
type IsolationForestConfig struct {
	NumTrees   int `mapstructure:"num_trees"`
	SampleSize int `mapstructure:"sample_size"`
}

// LOFConfig holds Local Outlier Factor parameters
type LOFConfig struct {
	K int `mapstructure:"k"` // Number of neighbors
}

// DBSCANConfig holds DBSCAN clustering parameters
type DBSCANConfig struct {
	Eps        float64 `mapstructure:"eps"`         // Maximum distance between points
	MinPts     int     `mapstructure:"min_pts"`     // Minimum points to form cluster
}

// ProspectivityAgentConfig holds prospectivity agent configuration
type ProspectivityAgentConfig struct {
	GridResolution float64            `mapstructure:"grid_resolution"` // meters
	Weights        map[string]float64 `mapstructure:"weights"`         // factor -> weight
}

// SamplingAgentConfig holds sampling optimization agent configuration
type SamplingAgentConfig struct {
	DefaultBudget   float64 `mapstructure:"default_budget"`
	CostPerSample   float64 `mapstructure:"cost_per_sample"`
	MaxSuggestions  int     `mapstructure:"max_suggestions"`
	GridResolution  float64 `mapstructure:"grid_resolution"`
}

// OrchestratorAgentConfig holds decision orchestrator configuration
type OrchestratorAgentConfig struct {
	MinConfidence      float64 `mapstructure:"min_confidence"`
	RequireHumanReview bool    `mapstructure:"require_human_review"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	JWTSecret      string   `mapstructure:"jwt_secret"`
	SignupPasskey  string   `mapstructure:"signup_passkey"`
	APIKeyHeader   string   `mapstructure:"api_key_header"`
	EnableAuth     bool     `mapstructure:"enable_auth"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// LLMConfig holds LLM API configuration for report generation
type LLMConfig struct {
	Provider string `mapstructure:"provider"` // openai, anthropic, etc.
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	MaxTokens int   `mapstructure:"max_tokens"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	v.SetDefault("mongodb.database", "GNOSISAGENT")
	v.SetDefault("mongodb.timeout", 10)
	v.SetDefault("agents.qc.outlier_threshold", 3.0)
	v.SetDefault("agents.qc.min_sample_size", 30)
	v.SetDefault("agents.qc.enable_lab_comparison", true)
	v.SetDefault("agents.anomaly.isolation_forest.num_trees", 100)
	v.SetDefault("agents.anomaly.isolation_forest.sample_size", 256)
	v.SetDefault("agents.anomaly.lof.k", 20)
	v.SetDefault("agents.anomaly.dbscan.eps", 0.5)
	v.SetDefault("agents.anomaly.dbscan.min_pts", 5)
	v.SetDefault("agents.prospectivity.grid_resolution", 100.0)
	v.SetDefault("agents.sampling.default_budget", 50000.0)
	v.SetDefault("agents.sampling.cost_per_sample", 500.0)
	v.SetDefault("agents.sampling.max_suggestions", 20)
	v.SetDefault("agents.orchestrator.min_confidence", 0.7)
	v.SetDefault("agents.orchestrator.require_human_review", false)
	v.SetDefault("security.api_key_header", "X-API-Key")
	v.SetDefault("llm.provider", "gemini")
	v.SetDefault("llm.model", "gemini-2.5-flash")
	v.SetDefault("llm.max_tokens", 4000)
	v.SetDefault("llm.api_key", "[ENCRYPTION_KEY]")
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
	v.SetDefault("org.id", "default")
	v.SetDefault("org.name", "Admin_Organization")
	v.SetDefault("org.deposit_models", []string{"epithermal", "porphyry", "orogenic"})

	// Read from config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Override with environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("GNOSISAGENT")
	
	// Bind top-level ENV variables directly if they don't follow the section prefix
	v.BindEnv("logging.level", "LOG_LEVEL")
	v.BindEnv("logging.format", "LOG_FORMAT")

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
