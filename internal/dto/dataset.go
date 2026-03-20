package dto

type DatasetSummary struct {
	ID          string
	Name        string
	SampleCount int
	UploadedAt  string
	Status      string
	QCScore     float64
}

