package integration

import (
	"context"
	"testing"
	"time"

	"gnosis-agent/internal/agents"
	"gnosis-agent/internal/agents/anomaly"
	"gnosis-agent/internal/agents/orchestrator"
	"gnosis-agent/internal/agents/prospectivity"
	"gnosis-agent/internal/agents/qc"
	"gnosis-agent/internal/agents/sampling"
	"gnosis-agent/internal/models"
)

func TestCompleteAgentPipeline(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()
	
	// Create agent runtime
	runtime := agents.NewAgentRuntime(ctx)
	
	// Register all agents
	qcAgent := qc.NewQCAgent(&cfg.Agents.QC)
	anomalyAgent := anomaly.NewAnomalyAgent(&cfg.Agents.Anomaly)
	prospectivityAgent := prospectivity.NewProspectivityAgent(&cfg.Agents.Prospectivity)
	samplingAgent := sampling.NewSamplingAgent(&cfg.Agents.Sampling)
	orchestratorAgent := orchestrator.NewOrchestratorAgent(&cfg.Agents.Orchestrator)
	
	if err := runtime.Register(qcAgent); err != nil {
		t.Fatalf("Failed to register QC agent: %v", err)
	}
	if err := runtime.Register(anomalyAgent); err != nil {
		t.Fatalf("Failed to register anomaly agent: %v", err)
	}
	if err := runtime.Register(prospectivityAgent); err != nil {
		t.Fatalf("Failed to register prospectivity agent: %v", err)
	}
	if err := runtime.Register(samplingAgent); err != nil {
		t.Fatalf("Failed to register sampling agent: %v", err)
	}
	if err := runtime.Register(orchestratorAgent); err != nil {
		t.Fatalf("Failed to register orchestrator agent: %v", err)
	}
	
	// Create test dataset
	dataset := CreateTestDataset()
	
	// Step 1: Run QC Agent
	t.Log("Running QC Agent...")
	start := time.Now()
	qcResult, err := runtime.Execute(ctx, "qc_agent", dataset)
	if err != nil {
		t.Fatalf("QC Agent failed: %v", err)
	}
	t.Logf("QC Agent completed in %v", time.Since(start))
	
	qcResults, ok := qcResult.Output.(*models.QCResults)
	if !ok {
		t.Fatalf("Expected QCResults, got %T", qcResult.Output)
	}
	
	t.Logf("QC Overall Score: %.2f", qcResults.OverallQC)
	passedCount := 0
	for _, result := range qcResults.Results {
		if result.Passed {
			passedCount++
		}
	}
	t.Logf("Samples passed: %d/%d", passedCount, len(qcResults.Results))
	
	if qcResults.OverallQC < 0.5 {
		t.Error("Overall QC score too low")
	}
	
	// Step 2: Run Anomaly Detection Agent
	t.Log("Running Anomaly Detection Agent...")
	
	// Note: AnomalyAgent.Execute expects QCResults, but also needs ProcessedDataset
	// For integration, we'll call DetectAnomalies directly
	anomalyResults, err := anomalyAgent.DetectAnomalies(dataset, qcResults)
	if err != nil {
		t.Fatalf("Anomaly Agent failed: %v", err)
	}
	
	t.Logf("Anomalies detected: %d", len(anomalyResults.Results))
	t.Logf("Clusters found: %d", len(anomalyResults.Clusters))
	
	highAnomalies := 0
	for _, result := range anomalyResults.Results {
		if result.AnomalyScore > 0.7 {
			highAnomalies++
			t.Logf("High anomaly sample %s: score=%.2f", result.SampleID, result.AnomalyScore)
		}
	}
	
	if highAnomalies == 0 {
		t.Error("Expected at least one high anomaly sample")
	}
	
	// Step 3: Run Prospectivity Agent
	t.Log("Running Prospectivity Agent...")
	prospectivityResult, err := runtime.Execute(ctx, "prospectivity_agent", anomalyResults)
	if err != nil {
		t.Fatalf("Prospectivity Agent failed: %v", err)
	}
	
	prospectivityResults, ok := prospectivityResult.Output.(*models.ProspectivityResults)
	if !ok {
		t.Fatalf("Expected ProspectivityResults, got %T", prospectivityResult.Output)
	}
	
	t.Logf("Targets identified: %d", len(prospectivityResults.Targets))
	for i, target := range prospectivityResults.Targets {
		t.Logf("Target %d: ID=%s, Score=%.2f, Confidence=%.2f",
			i+1, target.TargetID, target.ProspectivityScore, target.Confidence)
	}
	
	if len(prospectivityResults.Targets) == 0 {
		t.Error("Expected at least one prospectivity target")
	}
	
	// Step 4: Run Sampling Agent
	t.Log("Running Sampling Agent...")
	samplingResult, err := runtime.Execute(ctx, "sampling_agent", prospectivityResults)
	if err != nil {
		t.Fatalf("Sampling Agent failed: %v", err)
	}
	
	samplingPlan, ok := samplingResult.Output.(*models.SamplingPlan)
	if !ok {
		t.Fatalf("Expected SamplingPlan, got %T", samplingResult.Output)
	}
	
	t.Logf("Sampling recommendations: %d", len(samplingPlan.Recommendations))
	t.Logf("Total cost: $%.2f", samplingPlan.TotalCost)
	t.Logf("Expected ROI: %.2f", samplingPlan.ExpectedROI)
	
	// Step 5: Run Orchestrator Agent
	t.Log("Running Orchestrator Agent...")
	decisionInput := &orchestrator.DecisionInput{
		QCResults:            qcResults,
		AnomalyResults:       anomalyResults,
		ProspectivityResults: prospectivityResults,
		SamplingPlan:         samplingPlan,
	}
	
	orchestratorResult, err := runtime.Execute(ctx, "orchestrator_agent", decisionInput)
	if err != nil {
		t.Fatalf("Orchestrator Agent failed: %v", err)
	}
	
	decisions, ok := orchestratorResult.Output.(*models.DrillDecisions)
	if !ok {
		t.Fatalf("Expected DrillDecisions, got %T", orchestratorResult.Output)
	}
	
	t.Logf("Decisions made: %d", len(decisions.Decisions))
	
	drillCount := 0
	noDrillCount := 0
	moreDataCount := 0
	
	for _, decision := range decisions.Decisions {
		t.Logf("Decision for %s: %s (confidence: %.2f)", 
			decision.TargetID, decision.Decision, decision.Confidence)
		t.Logf("  Rationale: %v", decision.Rationale)
		
		switch decision.Decision {
		case "drill":
			drillCount++
		case "no_drill":
			noDrillCount++
		case "needs_more_data":
			moreDataCount++
		}
	}
	
	t.Logf("Decision summary: drill=%d, no_drill=%d, needs_more_data=%d",
		drillCount, noDrillCount, moreDataCount)
	
	// Verify decisions have rationale
	for _, decision := range decisions.Decisions {
		if len(decision.Rationale) == 0 {
			t.Errorf("Decision %s has no rationale", decision.DecisionID)
		}
		if decision.Confidence < 0 || decision.Confidence > 1 {
			t.Errorf("Invalid confidence for %s: %.2f", decision.DecisionID, decision.Confidence)
		}
	}
	
	t.Log("✓ Complete agent pipeline executed successfully")
}

func TestPipelinePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	cfg := getTestConfig()
	ctx := context.Background()
	
	// Create larger dataset
	dataset := CreateLargeTestDataset(100)
	
	// Setup agents
	runtime := agents.NewAgentRuntime(ctx)
	qcAgent := qc.NewQCAgent(&cfg.Agents.QC)
	runtime.Register(qcAgent)
	
	// Measure QC performance
	start := time.Now()
	_, err := runtime.Execute(ctx, "qc_agent", dataset)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("QC failed: %v", err)
	}
	
	t.Logf("QC processed %d samples in %v (%.2f samples/sec)",
		len(dataset.ProcessedAssays), duration,
		float64(len(dataset.ProcessedAssays))/duration.Seconds())
	
	// Performance threshold: should process at least 10 samples/second
	if duration.Seconds() > float64(len(dataset.ProcessedAssays))/10 {
		t.Errorf("QC performance below threshold: took %v for %d samples",
			duration, len(dataset.ProcessedAssays))
	}
}

func TestPipelineErrorHandling(t *testing.T) {
	cfg := getTestConfig()
	ctx := context.Background()
	
	runtime := agents.NewAgentRuntime(ctx)
	qcAgent := qc.NewQCAgent(&cfg.Agents.QC)
	runtime.Register(qcAgent)
	
	// Test with invalid input
	_, err := runtime.Execute(ctx, "qc_agent", "invalid input")
	if err == nil {
		t.Error("Expected error with invalid input")
	}
	
	// Test with non-existent agent
	_, err = runtime.Execute(ctx, "nonexistent_agent", nil)
	if err == nil {
		t.Error("Expected error with non-existent agent")
	}
}
