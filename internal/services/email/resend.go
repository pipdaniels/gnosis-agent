package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

// ResendService manages email dispatches via the Resend API
type ResendService struct {
	APIKey string
}

// NewResendService initializes a new Resend email boundary
func NewResendService() *ResendService {
	key := os.Getenv("RESEND_API_KEY")
	if key == "" {
		// Mock key for local dev bounds if missing
		key = "re_mock_key"
	}
	return &ResendService{
		APIKey: key,
	}
}

// SendOrganizationInvite triggers the invite mapping via Resend API
func (s *ResendService) SendOrganizationInvite(toEmail, orgID, inviteToken, originHost string) error {
	inviteURL := fmt.Sprintf("%s/invite?token=%s&org_id=%s", originHost, inviteToken, orgID)

	htmlBody := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
			<h2 style="color: #111827;">You've been invited!</h2>
			<p style="color: #4b5563; line-height: 1.5;">You have been invited to join an organization on the Geochem Agent platform.</p>
			<p style="color: #4b5563; line-height: 1.5;"><strong>Organization ID:</strong> %s</p>
			
			<div style="margin: 30px 0;">
				<a href="%s" style="background-color: #1a73e8; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 500;">Accept Invitation</a>
			</div>
			
			<p style="color: #6b7280; font-size: 0.85rem;">If you did not expect this invitation, you can safely ignore this email.</p>
		</div>
	`, orgID, inviteURL)

	payload := map[string]interface{}{
		"from":    "Acme <onboarding@resend.dev>",
		"to":      []string{toEmail},
		"subject": "Invitation to join Geochem Agent",
		"html":    htmlBody,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to build resend request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to dispatch resend logic: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("resend returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}
