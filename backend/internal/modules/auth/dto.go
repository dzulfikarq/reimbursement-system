package auth

import "time"

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type UserResponse struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Role           string     `json:"role"`
	DepartmentID   *string    `json:"department_id,omitempty"`
	DepartmentName *string    `json:"department_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
