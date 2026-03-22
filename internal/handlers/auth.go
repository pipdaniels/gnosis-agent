package handlers

import (
	"fmt"
	"net/http"
	"time"

	"gnosis-agent/internal/services/auth"
	"gnosis-agent/internal/web/templates/pages"
	"gnosis-agent/internal/web/templates/components"
	"gnosis-agent/internal/dto"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	service *auth.AuthService
}

func NewAuthHandler(service *auth.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// SignupPage renders the signup page
func (h *AuthHandler) SignupPage(c echo.Context) error {
	return render(c, pages.Signup())
}

// SigninPage renders the signin page
func (h *AuthHandler) SigninPage(c echo.Context) error {
	return render(c, pages.Signin())
}

// HandleSignup processes signup requests
func (h *AuthHandler) HandleSignup(c echo.Context) error {
	orgName := c.FormValue("org_name")
	email := c.FormValue("email")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirm_password")
	terms := c.FormValue("terms") == "on"

	req := dto.SignupRequest{
		OrgName:         orgName,
		Email:           email,
		Password:        password,
		ConfirmPassword: confirmPassword,
		TermsAccepted:   terms,
	}

	result, err := h.service.Signup(c.Request().Context(), req)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return render(c, components.Toast("Signup Failed", err.Error(), components.ToastError))
		}
		return c.String(http.StatusBadRequest, fmt.Sprintf("Signup failed: %v", err))
	}

	// Set Auth Cookie
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = result.Token
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Path = "/"
	c.SetCookie(cookie)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusFound, "/dashboard")
}

// HandleSignin processes signin requests
func (h *AuthHandler) HandleSignin(c echo.Context) error {
	orgID := c.FormValue("org_id")
	email := c.FormValue("email")
	password := c.FormValue("password")

	req := dto.LoginRequest{
		OrgID:    orgID,
		Email:    email,
		Password: password,
	}

	token, err := h.service.Login(c.Request().Context(), req)
	if err != nil {
		if c.Request().Header.Get("HX-Request") == "true" {
			return render(c, components.Toast("Signin Failed", "Invalid credentials", components.ToastError))
		}
		return c.String(http.StatusUnauthorized, "Invalid credentials")
	}

	// Set Auth Cookie
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = token
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Path = "/"
	c.SetCookie(cookie)

	if c.Request().Header.Get("HX-Request") == "true" {
		c.Response().Header().Set("HX-Redirect", "/dashboard")
		return c.NoContent(http.StatusOK)
	}
	return c.Redirect(http.StatusFound, "/dashboard")
}

// Logout clears the session
func (h *AuthHandler) Logout(c echo.Context) error {
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = ""
	cookie.Expires = time.Now().Add(-1 * time.Hour)
	cookie.HttpOnly = true
	cookie.Path = "/"
	c.SetCookie(cookie)
	return c.Redirect(http.StatusFound, "/signin")
}
