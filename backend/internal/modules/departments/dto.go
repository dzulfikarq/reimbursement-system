package departments

import "encoding/json"

// Budget rides as json.Number end-to-end — no float64 ever touches the money
// path (docs/03).
type CreateRequest struct {
	Name          string       `json:"name" binding:"required,max=100"`
	MonthlyBudget *json.Number `json:"monthly_budget" binding:"omitempty,numeric,gt=0"`
}

type UpdateRequest = CreateRequest

type DepartmentResponse struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	MonthlyBudget *json.Number `json:"monthly_budget,omitempty"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
}
