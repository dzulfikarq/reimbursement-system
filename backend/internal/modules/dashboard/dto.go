package dashboard

type SummaryResponse struct {
	PendingCount int64    `json:"pending_count"`
	MonthlyTotal string   `json:"monthly_total"`
	ApprovalRate *float64 `json:"approval_rate"`
}

type TrendPoint struct {
	Month string `json:"month"` // YYYY-MM
	Total string `json:"total"`
}

type TrendResponse struct {
	Months int          `json:"months"`
	Series []TrendPoint `json:"series"`
}

type BreakdownItem struct {
	CategoryID   string `json:"category_id"`
	CategoryName string `json:"category_name"`
	Total        string `json:"total"`
	ClaimCount   int64  `json:"claim_count"`
}

type BreakdownResponse struct {
	Month    string           `json:"month"`
	Items    []BreakdownItem  `json:"items"`
}
