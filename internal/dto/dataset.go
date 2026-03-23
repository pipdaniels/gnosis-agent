package dto

type DatasetSummary struct {
	ID          string
	Name        string
	SampleCount int
	UploadedAt  string
	Status      string
	QCScore     float64
}

// DatasetDetailDTO carries a raw dataset with its QC context
type DatasetDetailDTO struct {
	ID              string
	DatasetID       string
	ProjectName     string
	SamplingMethod  string
	LabName         string
	DepositType     string
	UploadedBy      string
	UploadedByEmail string
	UploadedAt      string
	SampleCount     int
	// FlaggedSampleIDs is a set for O(1) lookup in the template
	FlaggedSampleIDs map[string]bool
	// QCSummary maps flag type -> count
	QCSummary    map[string]int
	HasQC        bool
	OverallQC    float64
	TotalFlagged int
}

// MapPoint represents a single data point on the map
type MapPoint struct {
	SampleID string             `json:"sampleID"`
	Lat      float64            `json:"lat"`
	Lon      float64            `json:"lon"`
	QCStatus string             `json:"qcStatus"` // "passed", "flagged", "none"
	Elements map[string]float64 `json:"elements"`
}

// DatasetMapDTO holds metadata and points for the map view
type DatasetMapDTO struct {
	ID          string     `json:"id"`
	DatasetID   string     `json:"datasetID"`
	ProjectName string     `json:"projectName"`
	Points      []MapPoint `json:"points"`
}
