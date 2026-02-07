package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/pipdaniels/geochem-agent/internal/web/templates/pages"
)

func render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	return component.Render(c.Request().Context(), c.Response().Writer)
}

func Dashboard(c echo.Context) error {
	// TODO: Fetch real data
	// datasets []DatasetSummary, recentDecisions []DecisionSummary, stats map[string]interface{}
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

func Signup(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "Signup page not implemented yet")
}

func Signin(c echo.Context) error {
	return c.String(http.StatusNotImplemented, "Signin page not implemented yet")
}
