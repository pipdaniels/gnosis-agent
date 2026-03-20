package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/pipdaniels/geochem-agent/internal/db"
	"github.com/pipdaniels/geochem-agent/internal/models"
	"github.com/pipdaniels/geochem-agent/internal/services/ingestion"
	"github.com/pipdaniels/geochem-agent/internal/web/templates/pages"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func Dashboard(c echo.Context) error {
	// TODO: Fetch real data
	datasets := []pages.DatasetSummary{}
	decisions := []pages.DecisionSummary{}
	stats := map[string]interface{}{
		"targets": 0.0,
		"qc_rate": 0.0,
	}
	return render(c, pages.Dashboard(datasets, decisions, stats))
}

func Upload(c echo.Context) error {
	return render(c, pages.Upload())
}

// WebHandler holds dependencies for web-layer handlers.
type WebHandler struct {
	mongo *db.MongoManager
}

// NewWebHandler creates a WebHandler with the given MongoManager.
func NewWebHandler(mongo *db.MongoManager) *WebHandler {
	return &WebHandler{mongo: mongo}
}

// HandleUpload processes a multipart file upload (CSV or XLSX) and saves
// the parsed assays as a RawDataset in the organisation's raw_datasets collection.
func (h *WebHandler) HandleUpload(c echo.Context) error {
	// 1. Extract org / user from JWT claims set by echojwt middleware.
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing auth token"})
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token claims"})
	}
	orgID, _ := claims["org_id"].(string)
	userID, _ := claims["user_id"].(string)
	if orgID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "org_id not found in token"})
	}

	// 2. Parse multipart form (max 50 MB).
	if err := c.Request().ParseMultipartForm(50 << 20); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to parse form: " + err.Error()})
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "no file provided"})
	}

	// 3. Collect form metadata.
	projectName := c.FormValue("project_name")
	samplingMethod := c.FormValue("sampling_method")
	labName := c.FormValue("lab_name")
	depositType := c.FormValue("deposit_type")
	commodities := c.FormValue("commodities")
	samplingDateStr := c.FormValue("sampling_date")

	if projectName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "project_name is required"})
	}

	// Parse sampling date; fall back to now.
	collectedAt := time.Now()
	if samplingDateStr != "" {
		if t, parseErr := time.Parse("2006-01-02", samplingDateStr); parseErr == nil {
			collectedAt = t
		}
	}

	// 4. Open file and select parser based on extension.
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".csv" && ext != ".xlsx" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("unsupported file type %q; only .csv and .xlsx are accepted", ext),
		})
	}

	f, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open uploaded file"})
	}
	defer f.Close()

	var assays []models.Assay
	switch ext {
	case ".csv":
		assays, err = ingestion.ParseCSV(f, labName, collectedAt)
	case ".xlsx":
		assays, err = ingestion.ParseXLSX(f, labName, collectedAt)
	}
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "parse error: " + err.Error()})
	}

	// 5. Build RawDataset.
	datasetID := "ds_" + generateID(12)
	dataset := models.RawDataset{
		ID:        primitive.NewObjectID(),
		OrgID:     orgID,
		DatasetID: datasetID,
		RawAssays: assays,
		Metadata: models.DatasetMetadata{
			SamplingMethod: samplingMethod,
			ProjectName:    projectName,
			DepositType:    depositType,
			UploadedBy:     userID,
			Notes:          commodities,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// 6. Insert into MongoDB.
	orgDB, err := h.mongo.GetOrgDatabase(orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "database error: " + err.Error()})
	}

	ctx := c.Request().Context()
	if _, err := orgDB.Collection("raw_datasets").InsertOne(ctx, dataset); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save dataset: " + err.Error()})
	}

	// 7. Return success with dataset info.
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":         dataset.ID.Hex(),
		"dataset_id": dataset.DatasetID,
		"samples":    len(assays),
	})
}

// generateID returns a random URL-safe string of length n.
func generateID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}
