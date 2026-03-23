package dto

// LoginRequest body represents the data needeed for login
type LoginRequest struct {
	OrgID    string `json:"org_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents the data body returned after a successful login
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// SignupRequest represents the data needed for signup
type SignupRequest struct {
	OrgName         string `json:"org_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	TermsAccepted   bool   `json:"terms_accepted"`
}

// SignupResult contains the result of a successful signup
type SignupResult struct {
	OrgID  string
	Token  string
	APIKey string
}
