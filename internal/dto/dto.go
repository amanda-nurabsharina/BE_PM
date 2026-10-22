package dto

// Auth DTOs
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type QuickLoginRequest struct {
	Email string `json:"email"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

// Task DTOs
type TaskCreateRequest struct {
	Type           string   `json:"type"` // endorse | barter | other
	Title          string   `json:"title"`
	Brand          string   `json:"brand"`
	FeeOrValue     int64    `json:"fee_or_value"`
	DueDate        string   `json:"due_date"` // YYYY-MM-DD
	Deliverable    string   `json:"deliverable"`
	Platform       string   `json:"platform"`
	ShipmentNumber string   `json:"shipment_number"`
	PicName        string   `json:"pic_name"`
	PicPhone       string   `json:"pic_phone"`
	Priority       string   `json:"priority"` // high | medium | low
	Status         string   `json:"status"`   // initial status
	Brief          string   `json:"brief"`
	Checklist      []string `json:"checklist"`
}

type TaskStatusUpdateRequest struct {
	Status string `json:"status"`
}

type PaymentConfirmationRequest struct {
	PaymentProofURL  string `json:"payment_proof_url"`
	PaymentProofName string `json:"payment_proof_name"`
}

// Invoice DTOs
type InvoiceItemRequest struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
}

type InvoiceCreateRequest struct {
	RelatedTaskID *uint                `json:"related_task_id"`
	ClientName    string               `json:"client_name"`
	ClientEmail   string               `json:"client_email"`
	ClientPhone   string               `json:"client_phone"`
	ClientAddress string               `json:"client_address"`
	DueDate       string               `json:"due_date"` // YYYY-MM-DD
	IncludeTax    bool                 `json:"include_tax"`
	Items         []InvoiceItemRequest `json:"items"`
}

// Cashflow DTOs
type TransactionCreateRequest struct {
	Type           string `json:"type"` // income | expense
	Amount         int64  `json:"amount"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	Date           string `json:"date"` // YYYY-MM-DD
	AttachmentURL  string `json:"attachment_url"`
	AttachmentName string `json:"attachment_name"`
}

// Settings DTOs
type SettingsUpdateRequest struct {
	Name              string  `json:"name"`
	Company           string  `json:"company"`
	Phone             string  `json:"phone"`
	CompanyAddress    string  `json:"company_address"`
	NPWP              string  `json:"npwp"`
	TaxRate           float64 `json:"tax_rate"`
	LogoURL           string  `json:"logo_url"`
	SignatureURL      string  `json:"signature_url"`
	BankName          string  `json:"bank_name"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankAccountHolder string  `json:"bank_account_holder"`
	AvatarURL         string  `json:"avatar_url"`
}

// Dashboard DTO
type DashboardResponse struct {
	EstimatedTotalValue int64       `json:"estimated_total_value"`
	RealCashBalance     int64       `json:"real_cash_balance"`
	ActiveTasksCount    int         `json:"active_tasks_count"`
	UnpaidInvoicesCount int         `json:"unpaid_invoices_count"`
	UnpaidInvoicesTotal int64       `json:"unpaid_invoices_total"`
	CashflowThisMonth   CashflowSum `json:"cashflow_this_month"`
	TopAgendaTasks      interface{} `json:"top_agenda_tasks"`
	PendingInvoices     interface{} `json:"pending_invoices"`
}

type CashflowSum struct {
	TotalIncome  int64 `json:"total_income"`
	TotalExpense int64 `json:"total_expense"`
	NetCashflow  int64 `json:"net_cashflow"`
}

// Reports DTO
type ReportResponse struct {
	PeriodLabel     string      `json:"period_label"`
	FilterType      string      `json:"filter_type"`
	TotalProjects   int         `json:"total_projects"`
	TotalFeeRevenue int64       `json:"total_fee_revenue"`
	TotalPaid       int64       `json:"total_paid"`
	TotalUnpaid     int64       `json:"total_unpaid"`
	BarterValuation int64       `json:"barter_valuation"`
	Projects        interface{} `json:"projects"`
	Pagination      PageMeta    `json:"pagination"`
}

type PageMeta struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

type DatePeriodOption struct {
	Month int    `json:"month"`
	Year  int    `json:"year"`
	Label string `json:"label"` // e.g. "Oktober 2026"
}
