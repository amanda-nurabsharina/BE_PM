package repository

import (
	"fmt"
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

	oct2026 := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	// Seed Tasks
	task1 := domain.Task{
		UserID:         user.ID,
		Type:           "endorse",
		Title:          "Endorse TikTok Somethinc Niacinamide Serum",
		Brand:          "Somethinc Official",
		FeeOrValue:     10000000,
		DueDate:        oct2026.AddDate(0, 0, 27), // 2026-10-28
		Deliverable:    "1x TikTok Video 60s + Keranjang Kuning",
		Priority:       "high",
		Status:         "waiting_payment",
		Brief:          "Fokus pada tekstur ringan dan uji pemakaian selama 7 hari. Sertakan link affiliate di keranjang kuning.",
		PicName:        "Sarah - Somethinc PIC",
		PicPhone:       "+62 811-2345-6789",
		Checklist: []domain.ChecklistItem{
			{Text: "Penerimaan Produk Sample & Brief", Completed: true},
			{Text: "Produksi Video Draft & Review Agensi", Completed: true},
			{Text: "Tayang Konten TikTok & Sematkan Keranjang", Completed: true},
			{Text: "Kirim Laporan Analytics Insights 24 Jam", Completed: false},
		},
	}
	r.db.Create(&task1)

	task2 := domain.Task{
		UserID:         user.ID,
		Type:           "barter",
		Title:          "Barter Review Erha Acne Act Spot Gel",
		Brand:          "Erha Skincare",
		FeeOrValue:     650000,
		DueDate:        oct2026.AddDate(0, 0, 25), // 2026-10-26
		Platform:       "TikTok & IG Reels",
		ShipmentNumber: "JNT-882910221",
		Priority:       "medium",
		Status:         "draft",
		Brief:          "Review jujur pemakaian 3 hari untuk jerawat aktif. Tonjolkan kandungan salicylic acid.",
		PicName:        "Rian Erha",
		PicPhone:       "+62 812-9988-7766",
		Checklist: []domain.ChecklistItem{
			{Text: "Cek Resi & Unboxing Sample", Completed: true},
			{Text: "Ambil Footage Before/After", Completed: false},
			{Text: "Upload Reels & Mention Official", Completed: false},
		},
	}
	r.db.Create(&task2)

	task3 := domain.Task{
		UserID:           user.ID,
		Type:             "endorse",
		Title:            "Endorse Scarlett Whitening Body Lotion 11.11",
		Brand:            "Scarlett Whitening",
		FeeOrValue:       8500000,
		DueDate:          oct2026.AddDate(0, 0, 14), // 2026-10-15
		Deliverable:      "1x IG Reels + 2 Story Promo 11.11",
		Priority:         "high",
		Status:           "paid",
		PaymentProofURL:  "https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80",
		PaymentProofName: "bukti_tf_bca_scarlett.jpg",
		PicName:          "Budi Scarlett",
		PicPhone:         "+62 812-4433-2211",
		Checklist: []domain.ChecklistItem{
			{Text: "Shooting Reels Konsep OOTD", Completed: true},
			{Text: "Approval Script & Music Trending", Completed: true},
			{Text: "Post Reels & Stories Swipe Up", Completed: true},
		},
	}
	r.db.Create(&task3)

	task4 := domain.Task{
		UserID:         user.ID,
		Type:           "other",
		Title:          "Live Streaming Shopee 11.11 Mega Sale (3 Jam Non-stop)",
		Brand:          "Internal Studio",
		FeeOrValue:     0,
		DueDate:        time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC),
		Priority:       "high",
		Status:         "draft",
		Brief:          "Live streaming session khusus studio affiliate untuk target omset GMV 100 Juta.",
		Checklist: []domain.ChecklistItem{
			{Text: "Setup Lighting Key & Rim Light", Completed: false},
			{Text: "Display 20 Produk Sample di Meja", Completed: false},
			{Text: "Briefing Co-Host & Rundown Promo Flash Sale", Completed: false},
		},
	}
	r.db.Create(&task4)

	// Seed Invoices
	inv1 := domain.Invoice{
		UserID:           user.ID,
		InvoiceNumber:    "INV/AFFILIATE/2026/101",
		ClientName:       "Shopee Affiliate Creator Program (PT Shopee International Indonesia)",
		ClientEmail:      "finance-affiliate@shopee.co.id",
		ClientPhone:      "+62 811-9876-5432",
		ClientAddress:    "Pacific Century Place Lt. 25, SCBD Jakarta Selatan",
		IssueDate:        oct2026.AddDate(0, 0, 4), // 2026-10-05
		DueDate:          oct2026.AddDate(0, 0, 19),
		IncludeTax:       true,
		TaxRate:          11.0,
		Subtotal:         17500000,
		TaxAmount:        1925000,
		GrandTotal:       19425000,
		Status:           "paid",
		PaymentProofURL:  "https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80",
		PaymentProofName: "bukti_tf_shopee_okt.jpg",
		Items: []domain.InvoiceItem{
			{Description: "Komisi Affiliate Campaign Shopee 10.10 Brand Day", Quantity: 1, UnitPrice: 17500000, Total: 17500000},
		},
	}
	r.db.Create(&inv1)

	inv2 := domain.Invoice{
		UserID:        user.ID,
		RelatedTaskID: &task1.ID,
		InvoiceNumber: "INV/ENDORSE/2026/102",
		ClientName:    "Somethinc Official (PT Beauty Haul Indonesia)",
		ClientEmail:   "finance@beautyhaul.com",
		ClientPhone:   "+62 811-2345-6789",
		ClientAddress: "Green Lake City Rukan Cordoba Blok E No. 8, Tangerang",
		IssueDate:     oct2026.AddDate(0, 0, 17), // 2026-10-18
		DueDate:       oct2026.AddDate(0, 0, 31),
		IncludeTax:    true,
		TaxRate:       11.0,
		Subtotal:      10000000,
		TaxAmount:     1100000,
		GrandTotal:    11100000,
		Status:        "unpaid",
		Items: []domain.InvoiceItem{
			{Description: "Paket Endorse Video 60s TikTok + Keranjang Kuning", Quantity: 1, UnitPrice: 10000000, Total: 10000000},
		},
	}
	r.db.Create(&inv2)
	task1.RelatedInvoiceID = &inv2.ID
	r.db.Save(&task1)

	inv3 := domain.Invoice{
		UserID:        user.ID,
		InvoiceNumber: "INV/ENDORSE/2026/103",
		ClientName:    "Erha Skincare Official",
		ClientEmail:   "ap-vendor@erha.co.id",
		ClientPhone:   "+62 812-9988-7766",
		ClientAddress: "Jl. Iskandar Muda No. 18, Kebayoran Lama, Jakarta Selatan",
		IssueDate:     oct2026.AddDate(0, 0, 21), // 2026-10-22
		DueDate:       oct2026.AddDate(0, 1, 5),
		IncludeTax:    false,
		TaxRate:       0.0,
		Subtotal:      3500000,
		TaxAmount:     0,
		GrandTotal:    3500000,
		Status:        "unpaid",
		Items: []domain.InvoiceItem{
			{Description: "Jasa Pembuatan Konten Dedicated UGC Spot Gel", Quantity: 1, UnitPrice: 3500000, Total: 3500000},
		},
	}
	r.db.Create(&inv3)

	inv4 := domain.Invoice{
		UserID:           user.ID,
		InvoiceNumber:    "INV/ENDORSE/2026/094",
		ClientName:       "Skintific Indonesia (PT Maycare Global)",
		ClientEmail:      "invoice@skintific.id",
		ClientPhone:      "+62 812-4433-2211",
		ClientAddress:    "AIA Central Lt. 18, Jl. Jend. Sudirman Kav. 48A, Jakarta",
		IssueDate:        time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
		DueDate:          time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC),
		IncludeTax:       true,
		TaxRate:          11.0,
		Subtotal:         12000000,
		TaxAmount:        1320000,
		GrandTotal:       13320000,
		Status:           "paid",
		PaymentProofURL:  "https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80",
		PaymentProofName: "bukti_transfer_skintific.jpg",
		Items: []domain.InvoiceItem{
			{Description: "Dedicated Review Cover All Perfect Cushion", Quantity: 1, UnitPrice: 12000000, Total: 12000000},
		},
	}
	r.db.Create(&inv4)

	// Seed Transactions (matching Mockup Cashflow: Pemasukan Rp 61.480.000, Pengeluaran Rp 8.914.000, Net Rp 52.566.000)
	tx1 := domain.Transaction{
		UserID:           user.ID,
		RelatedInvoiceID: &inv1.ID,
		Type:             "income",
		Amount:           19425000,
		Description:      "Pelunasan Invoice INV/AFFILIATE/2026/101 (Shopee Indonesia)",
		Category:         "Komisi Affiliate",
		Date:             oct2026.AddDate(0, 0, 19),
		AttachmentURL:    inv1.PaymentProofURL,
		AttachmentName:   inv1.PaymentProofName,
	}
	r.db.Create(&tx1)

	tx2 := domain.Transaction{
		UserID:           user.ID,
		RelatedInvoiceID: &inv4.ID,
		Type:             "income",
		Amount:           13320000,
		Description:      "Pelunasan Invoice INV/ENDORSE/2026/094 (Skintific Indonesia)",
		Category:         "Pembayaran Endorse",
		Date:             oct2026.AddDate(0, 0, 10),
		AttachmentURL:    inv4.PaymentProofURL,
		AttachmentName:   inv4.PaymentProofName,
	}
	r.db.Create(&tx2)

	tx3 := domain.Transaction{
		UserID:      user.ID,
		Type:        "income",
		Amount:      28735000,
		Description: "Bonus Performance Affiliate Brand Kolaborasi Q3",
		Category:    "Komisi Affiliate",
		Date:        oct2026.AddDate(0, 0, 5),
	}
	r.db.Create(&tx3)

	// Expenses (Total: Rp 8.914.000)
	exp1 := domain.Transaction{
		UserID:         user.ID,
		Type:           "expense",
		Amount:         3500000,
		Description:    "Sewa Kamera Sony FX3 & Lensa G-Master (3 Hari)",
		Category:       "Peralatan & Gear",
		Date:           oct2026.AddDate(0, 0, 12),
		AttachmentURL:  "https://images.unsplash.com/photo-1554224155-8d04cb21cd6c?w=600&auto=format&fit=crop&q=80",
		AttachmentName: "nota_rental_kamera.jpg",
	}
	r.db.Create(&exp1)

	exp2 := domain.Transaction{
		UserID:      user.ID,
		Type:        "expense",
		Amount:      2250000,
		Description: "Fee Freelance Video Editor 3 Video Endorse",
		Category:    "Tim & Editor",
		Date:        oct2026.AddDate(0, 0, 18),
	}
	r.db.Create(&exp2)

	exp3 := domain.Transaction{
		UserID:      user.ID,
		Type:        "expense",
		Amount:      1850000,
		Description: "Sewa Studio & Tambahan Lighting Aputure 300D",
		Category:    "Produksi Konten",
		Date:        oct2026.AddDate(0, 0, 20),
	}
	r.db.Create(&exp3)

	exp4 := domain.Transaction{
		UserID:      user.ID,
		Type:        "expense",
		Amount:      1314000,
		Description: "Konsumsi Crew & Transportasi Pengambilan Sample",
		Category:    "Operasional Lainnya",
		Date:        oct2026.AddDate(0, 0, 22),
	}
	r.db.Create(&exp4)

	// Seed Audit Logs
	logs := []domain.AuditLog{
		{UserID: user.ID, Actor: "Reza Winata", Module: "Setting", Action: "UPDATE_SETTINGS", Title: "Inisialisasi Pengaturan Studio", Detail: "Memperbarui identitas studio, NPWP, dan tarif PPN default 11%", CreatedAt: oct2026.AddDate(0, 0, 1)},
		{UserID: user.ID, Actor: "Reza Winata", Module: "Tugas", Action: "CREATE_TASK", Title: "Membuat Tugas Endorse Somethinc", Detail: "Tugas kampanye fee Rp 10.000.000 dibuat dengan status DRAFT", CreatedAt: oct2026.AddDate(0, 0, 2)},
		{UserID: user.ID, Actor: "Reza Winata", Module: "Invoice", Action: "CREATE_INVOICE", Title: "Penerbitan Invoice INV/AFFILIATE/2026/101", Detail: "Menerbitkan invoice Shopee Indonesia Rp 19.425.000", CreatedAt: oct2026.AddDate(0, 0, 4)},
		{UserID: user.ID, Actor: "Reza Winata", Module: "Keuangan", Action: "CREATE_EXPENSE", Title: "Pencatatan Biaya Rental Kamera", Detail: "Biaya operasional sewa gear kamera senilai Rp 3.500.000", CreatedAt: oct2026.AddDate(0, 0, 12)},
		{UserID: user.ID, Actor: "Reza Winata", Module: "Tugas", Action: "MARK_PAID", Title: "Pelunasan Tugas Scarlett Whitening", Detail: "Pelunasan tugas dan auto-generate invoice lunas Rp 9.435.000", CreatedAt: oct2026.AddDate(0, 0, 15)},
	}
	for _, l := range logs {
		r.db.Create(&l)
	}

	return nil
}
