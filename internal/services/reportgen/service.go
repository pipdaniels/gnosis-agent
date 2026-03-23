package reportgen

import (
	"context"
	"fmt"
	"time"

	"gnosis-agent/internal/db"
	"gnosis-agent/internal/llm"
	"gnosis-agent/internal/models"
	"gnosis-agent/internal/services/orgcontext"

	"go.mongodb.org/mongo-driver/bson"
)

// Service generates technical reports using Gemini AI
type Service struct {
	db         *db.MongoManager
	gemini     *llm.GeminiClient
	orgContext *orgcontext.OrgContextService
}

// AnalysisResults aggregates all analysis data
type AnalysisResults struct {
	OrgID                string
	DatasetID            string
	ProjectName          string
	TotalSamples         int
	QCPassRate           float64
	AnomalyCount         int
	HighPriorityTargets  int
	DrillRecommendations string
	Commodities          string
	GeologicalSetting    string
	AnalysisDate         time.Time
	OverallConfidence    float64
	QCResults            *models.QCResults
	AnomalyResults       *models.AnomalyResults
	ProspectivityResults *models.ProspectivityResults
	Decisions            *models.DrillDecisions
}

// NewService creates a new report generation service with Gemini
func NewService(ctx context.Context, dbManager *db.MongoManager, geminiAPIKey, model string, orgCtx *orgcontext.OrgContextService) (*Service, error) {
	geminiClient, err := llm.NewGeminiClient(ctx, geminiAPIKey, model)
	if err != nil {
		return nil, err
	}
	
	return &Service{
		db:         dbManager,
		gemini:     geminiClient,
		orgContext: orgCtx,
	}, nil
}

// GenerateReport creates a complete technical report using Gemini AI
func (s *Service) GenerateReport(ctx context.Context, orgID, datasetID string) (*models.TechnicalReport, error) {
	results, err := s.aggregateResults(ctx, orgID, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate results: %w", err)
	}
	
	summary, err := s.generateExecutiveSummary(ctx, results)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}
	
	sections, err := s.generateSections(ctx, results)
	if err != nil {
		return nil, fmt.Errorf("failed to generate sections: %w", err)
	}
	
	report := &models.TechnicalReport{
		ReportID:         fmt.Sprintf("RPT-%s-%d", datasetID, time.Now().Unix()),
		OrgID:            orgID,
		DatasetID:        datasetID,
		GeneratedAt:      time.Now(),
		Title:            fmt.Sprintf("Geochemical Analysis Report - %s", results.ProjectName),
		ExecutiveSummary: summary,
		Sections:         sections,
		Recommendations:  s.generateRecommendations(results),
		Metadata: models.ReportMetadata{
			Author:       "GNOSISAGENT AI",
			ProjectName:  results.ProjectName,
			TotalSamples: results.TotalSamples,
			AnalysisDate: results.AnalysisDate,
			Confidence:   results.OverallConfidence,
			Version:      "1.0",
		},
	}
	
	return s.saveReport(ctx, orgID, report)
}

func (s *Service) generateExecutiveSummary(ctx context.Context, results *AnalysisResults) (string, error) {
	systemPrompt := "You are a professional exploration geologist writing reports for mining executives."
	
	userPrompt := fmt.Sprintf(`Generate an executive summary for a geochemical exploration report.

DATA:
- Project: %s
- Samples: %d
- QC Pass: %.1f%%
- Anomalies: %d
- High-Priority Targets: %d
- Recommendation: %s
- Commodities: %s
- Setting: %s
- Confidence: %.1f%%

Write 3 paragraphs (250-350 words):
1. Survey scope and data quality
2. Key findings and anomalies
3. Drill recommendations with confidence

Use professional geological language. NO headings or preamble.`,
		results.ProjectName, results.TotalSamples, results.QCPassRate*100,
		results.AnomalyCount, results.HighPriorityTargets, results.DrillRecommendations,
		results.Commodities, results.GeologicalSetting, results.OverallConfidence*100)
	
	return s.gemini.GenerateWithSystemPrompt(ctx, systemPrompt, userPrompt)
}

func (s *Service) generateSections(ctx context.Context, results *AnalysisResults) ([]models.ReportSection, error) {
	sections := make([]models.ReportSection, 0)
	
	qcContent, _ := s.gemini.GenerateText(ctx, fmt.Sprintf(
		`Write 2-3 paragraphs on data quality for a geochem report. %d samples, %.1f%% pass rate, %.2f QC score. Technical geological language.`,
		results.TotalSamples, results.QCPassRate*100, results.QCResults.OverallQC))
	sections = append(sections, models.ReportSection{Title: "Data Quality", Content: qcContent})
	
	anomalyContent, _ := s.gemini.GenerateText(ctx, fmt.Sprintf(
		`Write 2-3 paragraphs on geochemical anomalies. %d anomalies, %d clusters, %s commodities, %s setting.`,
		results.AnomalyCount, len(results.AnomalyResults.Clusters), results.Commodities, results.GeologicalSetting))
	sections = append(sections, models.ReportSection{Title: "Anomalies", Content: anomalyContent})
	
	return sections, nil
}

func (s *Service) generateRecommendations(results *AnalysisResults) []string {
	recommendations := make([]string, 0)
	drillCount := 0
	for _, d := range results.Decisions.Decisions {
		if d.Decision == "drill" {
			drillCount++
		}
	}
	if drillCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Proceed with drilling at %d targets", drillCount))
	}
	return recommendations
}

func (s *Service) aggregateResults(ctx context.Context, orgID, datasetID string) (*AnalysisResults, error) {
	db, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	
	var qc models.QCResults
	db.Collection("qc_results").FindOne(ctx, bson.M{"dataset_id": datasetID}).Decode(&qc)
	
	var anom models.AnomalyResults
	db.Collection("anomaly_results").FindOne(ctx, bson.M{"dataset_id": datasetID}).Decode(&anom)
	
	var prosp models.ProspectivityResults
	db.Collection("prospectivity_results").FindOne(ctx, bson.M{"dataset_id": datasetID}).Decode(&prosp)
	
	var dec models.DrillDecisions
	db.Collection("decisions").FindOne(ctx, bson.M{"dataset_id": datasetID}).Decode(&dec)
	
	return &AnalysisResults{
		OrgID: orgID, DatasetID: datasetID, ProjectName: "Survey",
		TotalSamples: len(qc.Results), QCPassRate: qc.OverallQC,
		AnomalyCount: len(anom.Results), Commodities: "Au,Cu", GeologicalSetting: "Porphyry",
		QCResults: &qc, AnomalyResults: &anom, ProspectivityResults: &prosp, Decisions: &dec,
	}, nil
}

func (s *Service) saveReport(ctx context.Context, orgID string, report *models.TechnicalReport) (*models.TechnicalReport, error) {
	db, err := s.db.GetOrgDatabase(orgID)
	if err != nil {
		return nil, err
	}
	_, err = db.Collection("tech_reports").InsertOne(ctx, report)
	return report, err
}

func (s *Service) GetReport(ctx context.Context, orgID, reportID string) (*models.TechnicalReport, error) {
	db, _ := s.db.GetOrgDatabase(orgID)
	var report models.TechnicalReport
	err := db.Collection("tech_reports").FindOne(ctx, bson.M{"report_id": reportID}).Decode(&report)
	return &report, err
}
