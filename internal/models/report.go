package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TechnicalReport represents a generated technical report
type TechnicalReport struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ReportID         string             `bson:"report_id" json:"report_id"`
	OrgID            string             `bson:"org_id" json:"org_id"`
	DatasetID        string             `bson:"dataset_id" json:"dataset_id"`
	GeneratedAt      time.Time          `bson:"generated_at" json:"generated_at"`
	Title            string             `bson:"title" json:"title"`
	ExecutiveSummary string             `bson:"executive_summary" json:"executive_summary"`
	Sections         []ReportSection    `bson:"sections" json:"sections"`
	Visualizations   []Visualization    `bson:"visualizations" json:"visualizations"`
	Recommendations  []string           `bson:"recommendations" json:"recommendations"`
	Metadata         ReportMetadata     `bson:"metadata" json:"metadata"`
}

// ReportSection represents a section of a technical report
type ReportSection struct {
	Title      string  `bson:"title" json:"title"`
	Content    string  `bson:"content" json:"content"` // LLM-generated markdown
	DataTables []Table `bson:"data_tables" json:"data_tables"`
	Figures    []string `bson:"figures" json:"figures"` // Paths to image files
}

// Table represents a data table in a report
type Table struct {
	Caption string     `bson:"caption" json:"caption"`
	Headers []string   `bson:"headers" json:"headers"`
	Rows    [][]string `bson:"rows" json:"rows"`
}

// Visualization represents a chart or map visualization
type Visualization struct {
	Type        string `bson:"type" json:"type"` // "map", "chart", "histogram", etc.
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description" json:"description"`
	FilePath    string `bson:"file_path" json:"file_path"`
}

// ReportMetadata holds metadata about a report
type ReportMetadata struct {
	Author       string    `bson:"author" json:"author"`
	ProjectName  string    `bson:"project_name" json:"project_name"`
	TotalSamples int       `bson:"total_samples" json:"total_samples"`
	AnalysisDate time.Time `bson:"analysis_date" json:"analysis_date"`
	Confidence   float64   `bson:"confidence" json:"confidence"`
	Version      string    `bson:"version" json:"version"`
}
