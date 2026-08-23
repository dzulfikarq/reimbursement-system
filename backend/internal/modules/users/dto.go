package users

import (
	"time"
)

type CreateRequest struct {
	Name          string  `json:"name" binding:"required,max=100"`
	Email         string  `json:"email" binding:"required,email,max=255"`
	Password      string  `json:"password" binding:"required,min=8,max=72"`
	Role          string  `json:"role" binding:"required,oneof=employee manager finance admin"`
	DepartmentID  *string `json:"department_id" binding:"omitempty,uuid"`
}

type UpdateRequest struct {
	Name         *string `json:"name" binding:"omitempty,max=100"`
	Role         *string `json:"role" binding:"omitempty,oneof=employee manager finance admin"`
	DepartmentID *string `json:"department_id" binding:"omitempty,uuid"`
	IsActive     *bool   `json:"is_active"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type UserResponse struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	DepartmentID   *string   `json:"department_id,omitempty"`
	DepartmentName *string   `json:"department_name,omitempty"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}
