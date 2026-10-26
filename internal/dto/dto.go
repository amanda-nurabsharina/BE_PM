package dto

// Auth DTOs
type LoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
}

type QuickLoginRequest struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

// Tenant DTOs
type CreateTenantRequest struct {
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	OwnerName         string  `json:"owner_name"`
	OwnerEmail        string  `json:"owner_email"`
	OwnerPassword     string  `json:"owner_password"`
	Phone             string  `json:"phone"`
	CompanyAddress    string  `json:"company_address"`
	NPWP              string  `json:"npwp"`
	NoKTP             string  `json:"no_ktp"`
	TaxRate           float64 `json:"tax_rate"`
	BankName          string  `json:"bank_name"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankAccountHolder string  `json:"bank_account_holder"`
}

type TenantResponse struct {
	ID                uint    `json:"id"`
	Name              string  `json:"name"`
	Slug              string  `json:"slug"`
	Status            string  `json:"status"`
	LogoURL           string  `json:"logo_url"`
	Phone             string  `json:"phone"`
	CompanyAddress    string  `json:"company_address"`
	NPWP              string  `json:"npwp"`
	NoKTP             string  `json:"no_ktp"`
	TaxRate           float64 `json:"tax_rate"`
	BankName          string  `json:"bank_name"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankAccountHolder string  `json:"bank_account_holder"`
	OwnerID           uint    `json:"owner_id"`
	OwnerName         string  `json:"owner_name"`
	OwnerEmail        string  `json:"owner_email"`
	TotalUsers        int64   `json:"total_users"`
	TotalTasks        int64   `json:"total_tasks"`
	CreatedAt         string  `json:"created_at"`
}

type UpdateTenantStatusRequest struct {
	Status string `json:"status"` // active | suspended
}

type ResetTenantOwnerPasswordRequest struct {
	NewPassword string `json:"new_password"`
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
	AutoInvoice    *bool    `json:"auto_invoice"`
}

type TaskStatusUpdateRequest struct {
	Status string `json:"status"`
}

type PaymentConfirmationRequest struct {
	PaymentProofURL  string   `json:"payment_proof_url"`
	PaymentProofName string   `json:"payment_proof_name"`
	TaxRate          *float64 `json:"tax_rate"`
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
	NoKTP             string  `json:"no_ktp"`
	TaxRate           float64 `json:"tax_rate"`
	LogoURL           string  `json:"logo_url"`
	SignatureURL      string  `json:"signature_url"`
	BankName          string  `json:"bank_name"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankAccountHolder string  `json:"bank_account_holder"`
	AvatarURL         string  `json:"avatar_url"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UserCreateRequest struct {
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	Role         string   `json:"role"`
	AvatarURL    string   `json:"avatar_url"`
	AllowedMenus []string `json:"allowed_menus"`
}

type UserUpdateRequest struct {
	Name         string   `json:"name"`
	Email        string   `json:"email"`
	Password     string   `json:"password"`
	Role         string   `json:"role"`
	AvatarURL    string   `json:"avatar_url"`
	AllowedMenus []string `json:"allowed_menus"`
}

type UserDetailResponse struct {
	ID                uint        `json:"id"`
	TenantID          uint        `json:"tenant_id"`
	TenantSlug        string      `json:"tenant_slug"`
	TenantName        string      `json:"tenant_name"`
	IsOwner           bool        `json:"is_owner"`
	NPWP              string      `json:"npwp,omitempty"`
	NoKTP             string      `json:"no_ktp,omitempty"`
	Email             string      `json:"email"`
	Name              string      `json:"name"`
	Company           string      `json:"company,omitempty"`
	Phone             string      `json:"phone,omitempty"`
	CompanyAddress    string      `json:"company_address,omitempty"`
	TaxRate           *float64    `json:"tax_rate,omitempty"`
	LogoURL           string      `json:"logo_url,omitempty"`
	SignatureURL      string      `json:"signature_url,omitempty"`
	BankName          string      `json:"bank_name,omitempty"`
	BankAccountNumber string      `json:"bank_account_number,omitempty"`
	BankAccountHolder string      `json:"bank_account_holder,omitempty"`
	AvatarURL         string      `json:"avatar_url"`
	Role              string      `json:"role"`
	RoleName          string      `json:"role_name"`
	AllowedMenus      []string    `json:"allowed_menus"`
	OwnerID           *uint       `json:"owner_id"`
	Tenant            interface{} `json:"tenant,omitempty"`
	CreatedAt         string      `json:"created_at"`
}

type RoleRequest struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Menus       []string `json:"menus"`
}

type RoleResponse struct {
	ID          uint     `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Menus       []string `json:"menus"`
	IsSystem    bool     `json:"is_system"`
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

type ReportExpenseItem struct {
	Index       int    `json:"index"`
	ID          uint   `json:"id"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Amount      int64  `json:"amount"`
	Date        string `json:"date"`
}

// Reports DTO
type ReportResponse struct {
	PeriodLabel     string              `json:"period_label"`
	FilterType      string              `json:"filter_type"`
	TotalProjects   int                 `json:"total_projects"`
	TotalFeeRevenue int64               `json:"total_fee_revenue"`
	TotalPaid       int64               `json:"total_paid"`
	TotalUnpaid     int64               `json:"total_unpaid"`
	BarterValuation int64               `json:"barter_valuation"`
	TotalExpense    int64               `json:"total_expense"`
	NetIncome       int64               `json:"net_income"`
	Expenses        []ReportExpenseItem `json:"expenses"`
	Projects        interface{}         `json:"projects"`
	Pagination      PageMeta            `json:"pagination"`
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
