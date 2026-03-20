package dto

import "time"

type OrganizationCreateRequest struct {
	Name      string    `json:"name"`
}

type OrganizationResponse struct {
	ID        string    `json:"org_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy User      `json:"created_by"`
	UpdatedBy User      `json:"updated_by"`
	Members   []User    `json:"members"`
}
		