package reimbursements

import "encoding/json"

type ItemRequest struct {
	Description string       `json:"description" binding:"required,max=200"`
	Quantity    int          `json:"quantity" binding:"required,gte=1,lte=10000"`
	UnitPrice   *json.Number `json:"unit_price" binding:"required,numeric,gt=0"`
}

type CreateRequest struct {
	Title        string        `json:"title" binding:"required,max=150"`
	CategoryID   string        `json:"category_id" binding:"required,uuid"`
	ExpenseDate  string        `json:"expense_date" binding:"required,datetime=2006-01-02"`
	Description  string        `json:"description" binding:"omitempty,max=2000"`
	Items        []ItemRequest `json:"items" binding:"required,min=1,max=50,dive"`
}

// Update replaces header fields + wholesale item replacement (simplest
// consistent model for drafts).
type UpdateRequest = CreateRequest

type RejectRequest struct {
	Note string `json:"note" binding:"required,min=3,max=1000"`
}

type ItemResponse struct {
	ID          string       `json:"id"`
	Description string       `json:"description"`
	Quantity    int          `json:"quantity"`
	UnitPrice   *json.Number `json:"unit_price"`
	LineTotal   *json.Number `json:"line_total"`
}

type AttachmentResponse struct {
	ID               string `json:"id"`
	OriginalFilename string `json:"original_filename"`
	MimeType         string `json:"mime_type"`
	SizeBytes        int64  `json:"size_bytes"`
	CreatedAt        string `json:"created_at"`
}

type ReimbursementResponse struct {
	ID           string       `json:"id"`
	EmployeeID   string       `json:"employee_id"`
	EmployeeName string       `json:"employee_name,omitempty"`
	CategoryID   string       `json:"category_id"`
	CategoryName string       `json:"category_name,omitempty"`
	CategoryCode string       `json:"category_code,omitempty"`
	Title        string       `json:"title"`
	Description  string       `json:"description,omitempty"`
	ExpenseDate  string       `json:"expense_date"`
	Amount       *json.Number `json:"amount"`
	Status       string       `json:"status"`
	CreatedAt    string       `json:"created_at"`
	UpdatedAt    string       `json:"updated_at"`
}

type ApprovalStepResponse struct {
	StepNumber   int     `json:"step_number"`
	ApproverRole string  `json:"approver_role"`
	Status       string  `json:"status"`
	Note         *string `json:"note,omitempty"`
}

type DetailResponse struct {
	ReimbursementResponse
	Items       []ItemResponse         `json:"items"`
	Attachments []AttachmentResponse   `json:"attachments"`
	Approvals   []ApprovalStepResponse `json:"approvals"`
}
