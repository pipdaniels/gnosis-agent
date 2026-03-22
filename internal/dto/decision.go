package dto

type DecisionSummary struct {
	ID         string
	TargetID   string
	Decision   string
	Confidence float64
	CreatedAt  string
}
