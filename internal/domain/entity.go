package domain

import (
	"time"

	"gorm.io/gorm"
)

// Tenant represents a multi-tenant workspace/studio
type Tenant struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Name              string         `gorm:"size:200;not null" json:"name"`
	Slug              string         `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Status            string         `gorm:"size:50;default:'active'" json:"status"` // active | suspended
	LogoURL           string         `gorm:"type:text" json:"logo_url"`
	Phone             string         `gorm:"size:50" json:"phone"`
	CompanyAddress    string         `gorm:"type:text" json:"company_address"`
	NPWP              string         `gorm:"size:50" json:"npwp"`
	NoKTP             string         `gorm:"size:50" json:"no_ktp"`
	TaxRate           float64        `gorm:"default:11.0" json:"tax_rate"`
	BankName          string         `gorm:"size:100" json:"bank_name"`
	BankAccountNumber string         `gorm:"size:100" json:"bank_account_number"`
	BankAccountHolder string         `gorm:"size:150" json:"bank_account_holder"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents Creator Owner Profile & Studio Configuration
type User struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	TenantID          uint           `gorm:"index;default:1;not null" json:"tenant_id"`
	Email             string         `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash      string         `gorm:"size:255;not null" json:"-"`
	Name              string         `gorm:"size:150;not null" json:"name"`
	AvatarURL         string         `gorm:"type:text" json:"avatar_url"`
	Company           string         `gorm:"size:200" json:"company"`
	Phone             string         `gorm:"size:50" json:"phone"`
	CompanyAddress    string         `gorm:"type:text" json:"company_address"`
	NPWP              string         `gorm:"size:50" json:"npwp"`
	NoKTP             string         `gorm:"size:50" json:"no_ktp"`
	TaxRate           float64        `gorm:"default:11.0" json:"tax_rate"` // Default PPN % (0 - 100)
	LogoURL           string         `gorm:"type:text" json:"logo_url"`
	SignatureURL      string         `gorm:"type:text" json:"signature_url"`
	BankName          string         `gorm:"size:100" json:"bank_name"`
	BankAccountNumber string         `gorm:"size:100" json:"bank_account_number"`
	BankAccountHolder string         `gorm:"size:150" json:"bank_account_holder"`
	Role              string         `gorm:"size:50;default:'SUPER_ADMIN'" json:"role"`
	AllowedMenus      string         `gorm:"type:text" json:"allowed_menus"` // Comma-separated: "beranda,tugas,laporan,invoice,keuangan,pengaturan"
	OwnerID           *uint          `gorm:"index" json:"owner_id"`         // Pointer to studio owner user ID
	IsOwner           bool           `gorm:"default:false" json:"is_owner"` // True if initial tenant owner
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// Role represents system role and its permitted menu slugs
type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Code        string    `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Description string    `gorm:"size:255" json:"description"`
	Menus       string    `gorm:"type:text;not null" json:"menus"` // "beranda,tugas,laporan,invoice,keuangan,pengaturan"
	IsSystem    bool      `gorm:"default:false" json:"is_system"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Task represents Endorse, Barter, or Internal Studio Task
type Task struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	TenantID         uint            `gorm:"index;default:1;not null" json:"tenant_id"`
	UserID           uint            `gorm:"index;not null" json:"user_id"`
	Type             string          `gorm:"size:50;not null" json:"type"` // endorse | barter | other
	Title            string          `gorm:"size:255;not null" json:"title"`
	Brand            string          `gorm:"size:150;not null" json:"brand"`
	FeeOrValue       int64           `gorm:"default:0" json:"fee_or_value"` // Rupiah integer (anti-floating drift)
	DueDate          time.Time       `gorm:"not null" json:"due_date"`
	Deliverable      string          `gorm:"type:text" json:"deliverable"`
	Platform         string          `gorm:"size:100" json:"platform"`
	ShipmentNumber   string          `gorm:"size:100" json:"shipment_number"`
	PicName          string          `gorm:"size:150" json:"pic_name"`
	PicPhone         string          `gorm:"size:50" json:"pic_phone"`
	Priority         string          `gorm:"size:20;default:'medium'" json:"priority"` // high | medium | low
	Status           string          `gorm:"size:50;default:'draft'" json:"status"`   // draft | waiting_payment | paid | posting | completed
	Brief            string          `gorm:"type:text" json:"brief"`
	PaymentProofURL  string          `gorm:"type:text" json:"payment_proof_url"`
	PaymentProofName string          `gorm:"size:255" json:"payment_proof_name"`
	RelatedInvoiceID *uint           `gorm:"index" json:"related_invoice_id"`
	Checklist        []ChecklistItem `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"checklist"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`
}

// ChecklistItem represents workflow step checklist item
type ChecklistItem struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TaskID    uint      `gorm:"index;not null" json:"task_id"`
	Text      string    `gorm:"size:255;not null" json:"text"`
	Completed bool      `gorm:"default:false" json:"completed"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Invoice represents official billing invoice
type Invoice struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	TenantID         uint           `gorm:"index;default:1;not null" json:"tenant_id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	RelatedTaskID    *uint          `gorm:"index" json:"related_task_id"`
	InvoiceNumber    string         `gorm:"size:100;uniqueIndex;not null" json:"invoice_number"`
	ClientName       string         `gorm:"size:200;not null" json:"client_name"`
	ClientEmail      string         `gorm:"size:150" json:"client_email"`
	ClientPhone      string         `gorm:"size:50" json:"client_phone"`
	ClientAddress    string         `gorm:"type:text" json:"client_address"`
	IssueDate        time.Time      `gorm:"not null" json:"issue_date"`
	DueDate          time.Time      `gorm:"not null" json:"due_date"`
	IncludeTax       bool           `gorm:"default:true" json:"include_tax"`
	TaxRate          float64        `gorm:"default:11.0" json:"tax_rate"` // Tax percentage, e.g. 11.0
	Subtotal         int64          `gorm:"default:0" json:"subtotal"`
	TaxAmount        int64          `gorm:"default:0" json:"tax_amount"`
	GrandTotal       int64          `gorm:"default:0" json:"grand_total"`
	Status           string         `gorm:"size:30;default:'unpaid'" json:"status"` // unpaid | paid
	PaymentProofURL  string         `gorm:"type:text" json:"payment_proof_url"`
	PaymentProofName string         `gorm:"size:255" json:"payment_proof_name"`
	Items            []InvoiceItem  `gorm:"foreignKey:InvoiceID;constraint:OnDelete:CASCADE" json:"items"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// InvoiceItem represents line item in an invoice
type InvoiceItem struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	InvoiceID   uint      `gorm:"index;not null" json:"invoice_id"`
	Description string    `gorm:"size:255;not null" json:"description"`
	Quantity    int       `gorm:"default:1" json:"quantity"`
	UnitPrice   int64     `gorm:"default:0" json:"unit_price"`
	Total       int64     `gorm:"default:0" json:"total"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Transaction represents Cashflow ledger entry (Income or Expense)
type Transaction struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	TenantID         uint           `gorm:"index;default:1;not null" json:"tenant_id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	RelatedInvoiceID *uint          `gorm:"uniqueIndex" json:"related_invoice_id"` // Unique constraint prevents duplicate payments
	Type             string         `gorm:"size:20;not null" json:"type"`          // income | expense
	Amount           int64          `gorm:"not null" json:"amount"`
	Description      string         `gorm:"type:text;not null" json:"description"`
	Category         string         `gorm:"size:100;not null" json:"category"`
	Date             time.Time      `gorm:"not null" json:"date"`
	AttachmentURL    string         `gorm:"type:text" json:"attachment_url"`
	AttachmentName   string         `gorm:"size:255" json:"attachment_name"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// AuditLog represents immutable system action log
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"index;default:1;not null" json:"tenant_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Actor     string    `gorm:"size:150;not null" json:"actor"`
	Module    string    `gorm:"size:50;not null" json:"module"` // Tugas | Invoice | Keuangan | Setting
	Action    string    `gorm:"size:100;not null" json:"action"`
	Title     string    `gorm:"size:255;not null" json:"title"`
	Detail    string    `gorm:"type:text" json:"detail"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// SequenceCounter tracks atomic sequence numbers per tenant, year and prefix
type SequenceCounter struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TenantID  uint      `gorm:"uniqueIndex:idx_tenant_year_prefix;default:1;not null" json:"tenant_id"`
	Year      int       `gorm:"uniqueIndex:idx_tenant_year_prefix;not null" json:"year"`
	Prefix    string    `gorm:"size:50;uniqueIndex:idx_tenant_year_prefix;not null" json:"prefix"`
	LastSeq   int       `gorm:"default:0" json:"last_seq"`
	UpdatedAt time.Time `json:"updated_at"`
}
