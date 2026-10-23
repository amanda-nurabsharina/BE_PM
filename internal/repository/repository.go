package repository

import (
	"fmt"
	"log"
	"os"
	"time"

	"be-pm/internal/domain"
	"be-pm/pkg/security"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	DB() *gorm.DB
	AutoMigrate() error
	SeedInitialData() error
	CleanSampleData() error

	// User / Settings
	GetUserByEmail(email string) (*domain.User, error)
	GetUserByID(id uint) (*domain.User, error)
	CreateUser(user *domain.User) error
	UpdateUser(user *domain.User) error
	DeleteUser(userID uint) error
	GetUsers(ownerID uint) ([]domain.User, error)
	GetQuickUsers() ([]domain.User, error)
	UpdatePassword(userID uint, newHash string) error

	// Roles
	GetRoles() ([]domain.Role, error)
	GetRoleByCode(code string) (*domain.Role, error)
	GetRoleByID(id uint) (*domain.Role, error)
	CreateRole(role *domain.Role) error
	UpdateRole(role *domain.Role) error
	DeleteRole(id uint) error

	// Tasks
	GetTasks(userID uint, month, year int, taskType, status string, page, limit int) ([]domain.Task, int64, error)
	GetTaskByID(userID, taskID uint) (*domain.Task, error)
	CreateTask(task *domain.Task) error
	UpdateTask(task *domain.Task) error
	DeleteTask(userID, taskID uint) error
	ToggleChecklistItem(userID, taskID, itemID uint) (*domain.ChecklistItem, error)

	// Invoices
	GetInvoices(userID uint, month, year int, status string, page, limit int) ([]domain.Invoice, int64, error)
	GetInvoiceByID(userID, invoiceID uint) (*domain.Invoice, error)
	CreateInvoice(invoice *domain.Invoice) error
	UpdateInvoice(invoice *domain.Invoice) error
	GetUnpaidInvoicesCount(userID uint) (int64, int64, error) // count, sum
	NextInvoiceSequence(tx *gorm.DB, year int, prefix string) (int, error)

	// Cashflow
	GetTransactions(userID uint, month, year int, txType string, page, limit int) ([]domain.Transaction, int64, error)
	GetCashflowSummary(userID uint, month, year int) (int64, int64, int64, error) // income, expense, net
	CreateTransaction(tx *domain.Transaction) error
	DeleteTransaction(userID, txID uint) error

	// Audit Logs
	GetAuditLogs(userID uint, module, search string, page, limit int) ([]domain.AuditLog, int64, error)
	CreateAuditLog(log *domain.AuditLog) error

	// Dynamic Date Options
	GetAvailableMonths(userID uint) ([]map[string]interface{}, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) DB() *gorm.DB {
	return r.db
}

func (r *gormRepository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&domain.User{},
		&domain.Role{},
		&domain.Task{},
		&domain.ChecklistItem{},
		&domain.Invoice{},
		&domain.InvoiceItem{},
		&domain.Transaction{},
		&domain.AuditLog{},
		&domain.SequenceCounter{},
	)
}

// User methods
func (r *gormRepository) GetUserByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *gormRepository) GetUserByID(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.First(&user, id).Error
	return &user, err
}

func (r *gormRepository) CreateUser(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *gormRepository) UpdateUser(user *domain.User) error {
	return r.db.Save(user).Error
}

func (r *gormRepository) DeleteUser(userID uint) error {
	return r.db.Delete(&domain.User{}, userID).Error
}

func (r *gormRepository) GetUsers(ownerID uint) ([]domain.User, error) {
	var users []domain.User
	err := r.db.Where("id = ? OR owner_id = ?", ownerID, ownerID).Order("id asc").Find(&users).Error
	return users, err
}

func (r *gormRepository) GetQuickUsers() ([]domain.User, error) {
	var users []domain.User
	err := r.db.Select("id, name, email, company, avatar_url, role, allowed_menus").Find(&users).Error
	return users, err
}

func (r *gormRepository) UpdatePassword(userID uint, newHash string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("password_hash", newHash).Error
}

// Role methods
func (r *gormRepository) GetRoles() ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.Order("id asc").Find(&roles).Error
	return roles, err
}

func (r *gormRepository) GetRoleByCode(code string) (*domain.Role, error) {
	var role domain.Role
	err := r.db.Where("code = ?", code).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *gormRepository) GetRoleByID(id uint) (*domain.Role, error) {
	var role domain.Role
	err := r.db.First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *gormRepository) CreateRole(role *domain.Role) error {
	return r.db.Create(role).Error
}

func (r *gormRepository) UpdateRole(role *domain.Role) error {
	return r.db.Save(role).Error
}

func (r *gormRepository) DeleteRole(id uint) error {
	return r.db.Where("id = ? AND is_system = false", id).Delete(&domain.Role{}).Error
}

// Tasks methods
func (r *gormRepository) GetTasks(userID uint, month, year int, taskType, status string, page, limit int) ([]domain.Task, int64, error) {
	var tasks []domain.Task
	var total int64

	q := r.db.Model(&domain.Task{}).Where("user_id = ?", userID)

	if taskType != "" && taskType != "all" && taskType != "semua" {
		q = q.Where("type = ?", taskType)
	}

	if status != "" && status != "all" && status != "semua" {
		if status == "pending" {
			q = q.Where("status = ?", "waiting_payment")
		} else {
			q = q.Where("status = ?", status)
		}
	}

	if year > 0 && month > 0 {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		q = q.Where("due_date >= ? AND due_date < ?", startDate, endDate)
	} else if year > 0 {
		startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(1, 0, 0)
		q = q.Where("due_date >= ? AND due_date < ?", startDate, endDate)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Preload("Checklist").Order("due_date desc, id desc").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, total, err
}

func (r *gormRepository) GetTaskByID(userID, taskID uint) (*domain.Task, error) {
	var task domain.Task
	err := r.db.Where("id = ? AND user_id = ?", taskID, userID).Preload("Checklist").First(&task).Error
	return &task, err
}

func (r *gormRepository) CreateTask(task *domain.Task) error {
	return r.db.Create(task).Error
}

func (r *gormRepository) UpdateTask(task *domain.Task) error {
	return r.db.Save(task).Error
}

func (r *gormRepository) DeleteTask(userID, taskID uint) error {
	return r.db.Where("id = ? AND user_id = ?", taskID, userID).Delete(&domain.Task{}).Error
}

func (r *gormRepository) ToggleChecklistItem(userID, taskID, itemID uint) (*domain.ChecklistItem, error) {
	var item domain.ChecklistItem
	// Ensure item belongs to task and task belongs to user
	var task domain.Task
	if err := r.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		return nil, err
	}

	if err := r.db.Where("id = ? AND task_id = ?", itemID, taskID).First(&item).Error; err != nil {
		return nil, err
	}

	item.Completed = !item.Completed
	err := r.db.Save(&item).Error
	return &item, err
}

// Invoices methods
func (r *gormRepository) GetInvoices(userID uint, month, year int, status string, page, limit int) ([]domain.Invoice, int64, error) {
	var invoices []domain.Invoice
	var total int64

	q := r.db.Model(&domain.Invoice{}).Where("user_id = ?", userID)

	if status != "" && status != "all" && status != "semua" {
		if status == "unpaid" || status == "pending" {
			q = q.Where("status != ?", "paid")
		} else {
			q = q.Where("status = ?", status)
		}
	}

	if year > 0 && month > 0 {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		q = q.Where("issue_date >= ? AND issue_date < ?", startDate, endDate)
	} else if year > 0 {
		startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(1, 0, 0)
		q = q.Where("issue_date >= ? AND issue_date < ?", startDate, endDate)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Preload("Items").Order("id desc").Offset(offset).Limit(limit).Find(&invoices).Error
	return invoices, total, err
}

func (r *gormRepository) GetInvoiceByID(userID, invoiceID uint) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := r.db.Where("id = ? AND user_id = ?", invoiceID, userID).Preload("Items").First(&inv).Error
	return &inv, err
}

func (r *gormRepository) CreateInvoice(invoice *domain.Invoice) error {
	return r.db.Create(invoice).Error
}

func (r *gormRepository) UpdateInvoice(invoice *domain.Invoice) error {
	return r.db.Save(invoice).Error
}

func (r *gormRepository) GetUnpaidInvoicesCount(userID uint) (int64, int64, error) {
	var count int64
	var sum int64
	err := r.db.Model(&domain.Invoice{}).Where("user_id = ? AND status != ?", userID, "paid").Count(&count).Error
	if err != nil {
		return 0, 0, err
	}

	row := r.db.Model(&domain.Invoice{}).Where("user_id = ? AND status != ?", userID, "paid").Select("COALESCE(SUM(grand_total), 0)").Row()
	_ = row.Scan(&sum)
	return count, sum, nil
}

func (r *gormRepository) NextInvoiceSequence(tx *gorm.DB, year int, prefix string) (int, error) {
	var counter domain.SequenceCounter
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("year = ? AND prefix = ?", year, prefix).
		FirstOrCreate(&counter, domain.SequenceCounter{Year: year, Prefix: prefix, LastSeq: 0}).Error
	if err != nil {
		return 0, err
	}

	counter.LastSeq++
	if err := tx.Save(&counter).Error; err != nil {
		return 0, err
	}
	return counter.LastSeq, nil
}

// Cashflow methods
func (r *gormRepository) GetTransactions(userID uint, month, year int, txType string, page, limit int) ([]domain.Transaction, int64, error) {
	var txs []domain.Transaction
	var total int64

	q := r.db.Model(&domain.Transaction{}).Where("user_id = ?", userID)

	if txType != "" && txType != "all" && txType != "semua" {
		q = q.Where("type = ?", txType)
	}

	if year > 0 && month > 0 {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		q = q.Where("date >= ? AND date < ?", startDate, endDate)
	} else if year > 0 {
		startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(1, 0, 0)
		q = q.Where("date >= ? AND date < ?", startDate, endDate)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Order("date desc, id desc").Offset(offset).Limit(limit).Find(&txs).Error
	return txs, total, err
}

func (r *gormRepository) GetCashflowSummary(userID uint, month, year int) (int64, int64, int64, error) {
	var income int64
	var expense int64

	q := r.db.Model(&domain.Transaction{}).Where("user_id = ?", userID)
	if year > 0 && month > 0 {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		q = q.Where("date >= ? AND date < ?", startDate, endDate)
	}

	_ = q.Where("type = ?", "income").Select("COALESCE(SUM(amount), 0)").Row().Scan(&income)
	_ = r.db.Model(&domain.Transaction{}).Where("user_id = ?", userID).
		Where(func() *gorm.DB {
			if year > 0 && month > 0 {
				startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				endDate := startDate.AddDate(0, 1, 0)
				return r.db.Where("date >= ? AND date < ?", startDate, endDate)
			}
			return r.db
		}()).
		Where("type = ?", "expense").Select("COALESCE(SUM(amount), 0)").Row().Scan(&expense)

	return income, expense, (income - expense), nil
}

func (r *gormRepository) CreateTransaction(tx *domain.Transaction) error {
	return r.db.Create(tx).Error
}

func (r *gormRepository) DeleteTransaction(userID, txID uint) error {
	// Must not be related to an invoice (BR-09)
	var t domain.Transaction
	if err := r.db.Where("id = ? AND user_id = ?", txID, userID).First(&t).Error; err != nil {
		return err
	}
	if t.RelatedInvoiceID != nil {
		return fmt.Errorf("transaksi pelunasan invoice tidak dapat dihapus")
	}
	return r.db.Delete(&t).Error
}

// Audit Logs methods
func (r *gormRepository) GetAuditLogs(userID uint, module, search string, page, limit int) ([]domain.AuditLog, int64, error) {
	var logs []domain.AuditLog
	var total int64

	q := r.db.Model(&domain.AuditLog{}).Where("user_id = ?", userID)

	if module != "" && module != "all" && module != "semua" {
		q = q.Where("module = ?", module)
	}

	if search != "" {
		pattern := "%" + search + "%"
		q = q.Where("title LIKE ? OR detail LIKE ? OR actor LIKE ?", pattern, pattern, pattern)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := q.Order("created_at desc, id desc").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func (r *gormRepository) CreateAuditLog(log *domain.AuditLog) error {
	return r.db.Create(log).Error
}

// Dynamic months from actual database dates
func (r *gormRepository) GetAvailableMonths(userID uint) ([]map[string]interface{}, error) {
	monthMap := make(map[string]map[string]interface{})

	// Month names in Indonesian
	monthNames := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	// Always ensure current month and year are available
	now := time.Now()
	nowKey := fmt.Sprintf("%d-%02d", now.Year(), now.Month())
	monthMap[nowKey] = map[string]interface{}{
		"month": int(now.Month()),
		"year":  now.Year(),
		"label": fmt.Sprintf("%s %d", monthNames[now.Month()], now.Year()),
	}

	// Query distinct months from tasks
	var taskDates []time.Time
	r.db.Model(&domain.Task{}).Where("user_id = ?", userID).Pluck("due_date", &taskDates)
	for _, d := range taskDates {
		key := fmt.Sprintf("%d-%02d", d.Year(), d.Month())
		if _, exists := monthMap[key]; !exists && d.Year() > 2000 {
			monthMap[key] = map[string]interface{}{
				"month": int(d.Month()),
				"year":  d.Year(),
				"label": fmt.Sprintf("%s %d", monthNames[d.Month()], d.Year()),
			}
		}
	}

	// Query distinct months from transactions
	var txDates []time.Time
	r.db.Model(&domain.Transaction{}).Where("user_id = ?", userID).Pluck("date", &txDates)
	for _, d := range txDates {
		key := fmt.Sprintf("%d-%02d", d.Year(), d.Month())
		if _, exists := monthMap[key]; !exists && d.Year() > 2000 {
			monthMap[key] = map[string]interface{}{
				"month": int(d.Month()),
				"year":  d.Year(),
				"label": fmt.Sprintf("%s %d", monthNames[d.Month()], d.Year()),
			}
		}
	}

	result := make([]map[string]interface{}, 0, len(monthMap))
	for _, v := range monthMap {
		result = append(result, v)
	}

	return result, nil
}

// SeedInitialData sets up roles, Reza Winata demo super admin user and all initial mock records matching SRS and mockups
func (r *gormRepository) SeedInitialData() error {
	// 1. Seed Default System Roles
	defaultRoles := []domain.Role{
		{
			Code:        "SUPER_ADMIN",
			Name:        "Super Admin",
			Description: "Akses penuh ke seluruh modul sistem dan pengelolaan pengguna/role",
			Menus:       "beranda,tugas,laporan,invoice,keuangan,pengaturan",
			IsSystem:    true,
		},
		{
			Code:        "ADMIN",
			Name:        "Admin Studio",
			Description: "Pengelolaan operasional tugas, invoice, kas, dan laporan",
			Menus:       "beranda,tugas,laporan,invoice,keuangan",
			IsSystem:    true,
		},
		{
			Code:        "FINANCE",
			Name:        "Finance / Keuangan",
			Description: "Fokus pengelolaan invoice penagihan dan pencatatan kas",
			Menus:       "beranda,invoice,keuangan,laporan",
			IsSystem:    false,
		},
		{
			Code:        "STAFF",
			Name:        "Staff Kreatif",
			Description: "Fokus eksekusi tugas, checklist deliverable, dan beranda",
			Menus:       "beranda,tugas",
			IsSystem:    false,
		},
		{
			Code:        "VIEWER",
			Name:        "Pengamat / Viewer",
			Description: "Akses pemantauan statistik beranda dan rekap laporan",
			Menus:       "beranda,laporan",
			IsSystem:    false,
		},
	}
	for _, dr := range defaultRoles {
		var existing domain.Role
		if err := r.db.Where("code = ?", dr.Code).First(&existing).Error; err != nil {
			r.db.Create(&dr)
		}
	}

	// 2. Synchronize existing users: ensure initial user is SUPER_ADMIN
	r.db.Model(&domain.User{}).Where("role = ? OR email = ?", "CREATOR_OWNER", "reza@studio.id").Updates(map[string]interface{}{
		"role":          "SUPER_ADMIN",
		"allowed_menus": "beranda,tugas,laporan,invoice,keuangan,pengaturan",
	})

	var count int64
	r.db.Model(&domain.User{}).Count(&count)
	if count > 0 {
		return nil
	}

	hashedPassword, _ := security.HashPassword("password123")

	user := domain.User{
		Email:             "reza@studio.id",
		PasswordHash:      hashedPassword,
		Name:              "Reza Winata",
		AvatarURL:         "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
		Company:           "Reza Studio (Affiliator Network)",
		Phone:             "+62 812-3456-7890",
		CompanyAddress:    "Studio Kreatif Lt. 3, Jl. Senopati No. 42, Jaksel",
		NPWP:              "09.254.678.9-012.000",
		TaxRate:           11.0,
		BankName:          "Bank Central Asia (BCA)",
		BankAccountNumber: "8830192831",
		BankAccountHolder: "Reza Winata",
		Role:              "SUPER_ADMIN",
		AllowedMenus:      "beranda,tugas,laporan,invoice,keuangan,pengaturan",
	}

	if err := r.db.Create(&user).Error; err != nil {
		return err
	}

	return nil
}

// CleanSampleData purges mock sample tasks, invoices, transactions, and audit logs.
// All user login accounts and system/custom roles are 100% preserved.
func (r *gormRepository) CleanSampleData() error {
	// 1. Break foreign key references between tasks and invoices
	r.db.Exec(`UPDATE tasks SET related_invoice_id = NULL WHERE brand IN ('Somethinc Official', 'Erha Skincare', 'Scarlett Whitening', 'Internal Studio')`)
	r.db.Exec(`UPDATE invoices SET related_task_id = NULL WHERE invoice_number LIKE 'INV/%/2026/%'`)

	// 2. Delete sample checklist items
	r.db.Exec(`DELETE FROM checklist_items WHERE task_id IN (
		SELECT id FROM tasks WHERE brand IN ('Somethinc Official', 'Erha Skincare', 'Scarlett Whitening', 'Internal Studio')
	)`)

	// 3. Delete sample invoice items
	r.db.Exec(`DELETE FROM invoice_items WHERE invoice_id IN (
		SELECT id FROM invoices WHERE invoice_number LIKE 'INV/%/2026/%'
	)`)

	// 4. Delete sample transactions
	r.db.Exec(`DELETE FROM transactions WHERE description LIKE '%INV/AFFILIATE%' 
		OR description LIKE '%INV/ENDORSE%' 
		OR description LIKE '%Bonus Performance%' 
		OR description LIKE '%Sewa Kamera%' 
		OR description LIKE '%Fee Freelance Video Editor%' 
		OR description LIKE '%Sewa Studio%' 
		OR description LIKE '%Konsumsi Crew%'`)

	// 5. Delete sample invoices
	r.db.Exec(`DELETE FROM invoices WHERE invoice_number LIKE 'INV/%/2026/%'`)

	// 6. Delete sample tasks
	r.db.Exec(`DELETE FROM tasks WHERE brand IN ('Somethinc Official', 'Erha Skincare', 'Scarlett Whitening', 'Internal Studio')`)

	// 7. Delete sample audit logs
	r.db.Exec(`DELETE FROM audit_logs WHERE title IN (
		'Inisialisasi Pengaturan Studio',
		'Membuat Tugas Endorse Somethinc',
		'Penerbitan Invoice INV/AFFILIATE/2026/101',
		'Pencatatan Biaya Rental Kamera',
		'Pelunasan Tugas Scarlett Whitening'
	)`)

	// If CLEAN_SAMPLE_DATA or CLEAN_ALL_DATA is set to "true" in environment,
	// also perform a full wipe of all operational tables while keeping users & roles.
	if os.Getenv("CLEAN_SAMPLE_DATA") == "true" || os.Getenv("CLEAN_ALL_DATA") == "true" {
		r.db.Exec("UPDATE tasks SET related_invoice_id = NULL")
		r.db.Exec("UPDATE invoices SET related_task_id = NULL")
		r.db.Exec("DELETE FROM checklist_items")
		r.db.Exec("DELETE FROM invoice_items")
		r.db.Exec("DELETE FROM transactions")
		r.db.Exec("DELETE FROM invoices")
		r.db.Exec("DELETE FROM tasks")
		r.db.Exec("DELETE FROM audit_logs")
		log.Println("[CLEANUP] Full operational data wipe executed (users and roles 100% preserved).")
	}

	log.Println("[CLEANUP] Sample data cleanup completed successfully. Login accounts and roles preserved.")
	return nil
}
