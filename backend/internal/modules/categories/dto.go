package categories

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type CreateRequest struct {
	Code                     string       `json:"code" binding:"required,max=20,ascii"`
	Name                     string       `json:"name" binding:"required,max=100"`
	MonthlyLimitPerEmployee  *json.Number `json:"monthly_limit_per_employee" binding:"omitempty,numeric,gt=0"`
}

type UpdateRequest struct {
	Name                     string       `json:"name" binding:"required,max=100"`
	MonthlyLimitPerEmployee  *json.Number `json:"monthly_limit_per_employee" binding:"omitempty,numeric,gt=0"`
	IsActive                 *bool        `json:"is_active"`
}

type CategoryResponse struct {
	ID                      string       `json:"id"`
	Code                    string       `json:"code"`
	Name                    string       `json:"name"`
	MonthlyLimitPerEmployee *json.Number `json:"monthly_limit_per_employee,omitempty"`
	IsActive                bool         `json:"is_active"`
	CreatedAt               string       `json:"created_at"`
	UpdatedAt               string       `json:"updated_at"`
}

// Model mirrors the categories table; code immutable after create.
type Category struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey"`
	Code                    string
	Name                    string
	MonthlyLimitPerEmployee *string `gorm:"column:monthly_limit_per_employee"`
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (Category) TableName() string { return "categories" }
