package service

import (
	"errors"
	"fmt"
	"math"
	"time"

	"be-pm/config"
	"be-pm/internal/domain"
	"be-pm/internal/dto"
	"be-pm/internal/repository"
	"be-pm/pkg/security"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service interface {
	// Auth
	Login(req dto.LoginRequest) (*dto.LoginResponse, error)
	QuickLogin(email string) (*dto.LoginResponse, error)
	GetQuickUsers() ([]domain.User, error)

	// Dashboard
	GetDashboard(userID uint) (*dto.DashboardResponse, error)

	// Tasks
	GetTasks(userID uint, month, year int, taskType, status string, page, limit int) (map[string]interface{}, error)
	GetTaskByID(userID, taskID uint) (*domain.Task, error)
	CreateTask(userID uint, req dto.TaskCreateRequest) (*domain.Task, error)
	UpdateTaskStatus(userID, taskID uint, status string) (*domain.Task, error)
	ToggleChecklistItem(userID, taskID, itemID uint) (*domain.ChecklistItem, error)
	MarkTaskPaid(userID, taskID uint, req dto.PaymentConfirmationRequest) error
	DeleteTask(userID, taskID uint) error

	// Invoices
	GetInvoices(userID uint, month, year int, status string, page, limit int) (map[string]interface{}, error)
	GetInvoiceByID(userID, invoiceID uint) (*domain.Invoice, error)
	CreateInvoice(userID uint, req dto.InvoiceCreateRequest) (*domain.Invoice, error)
	ToggleInvoiceTax(userID, invoiceID uint) (*domain.Invoice, error)
	MarkInvoicePaid(userID, invoiceID uint, req dto.PaymentConfirmationRequest) error

	// Cashflow
	GetCashflow(userID uint, month, year int, txType string, page, limit int) (map[string]interface{}, error)
	CreateTransaction(userID uint, req dto.TransactionCreateRequest) (*domain.Transaction, error)
	DeleteTransaction(userID, txID uint) error

	// Reports
	GetReports(userID uint, month, year int, coopType string, page, limit int) (*dto.ReportResponse, error)

	// Settings
	GetSettings(userID uint) (*domain.User, error)
	UpdateSettings(userID uint, req dto.SettingsUpdateRequest) (*domain.User, error)

	// Audit Logs
	GetAuditLogs(userID uint, module, search string, page, limit int) (map[string]interface{}, error)

	// Dynamic months
	GetAvailableMonths(userID uint) ([]map[string]interface{}, error)
}

type serviceImpl struct {
	repo repository.Repository
	cfg  *config.Config
}

func NewService(repo repository.Repository, cfg *config.Config) Service {
	return &serviceImpl{repo: repo, cfg: cfg}
}

// Auth
func (s *serviceImpl) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email dan password wajib diisi")
	}

	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil {
		return nil, errors.New("kredensial login tidak valid")
	}

	if !security.ComparePassword(req.Password, user.PasswordHash) {
		return nil, errors.New("kredensial login tidak valid")
	}

	token, err := security.GenerateToken(user.ID, user.Email, user.Name, user.Role, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{Token: token, User: user}, nil
}

func (s *serviceImpl) QuickLogin(email string) (*dto.LoginResponse, error) {
	if email == "" {
		email = "reza@studio.id"
	}

	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("profil tidak ditemukan")
	}

	token, err := security.GenerateToken(user.ID, user.Email, user.Name, user.Role, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{Token: token, User: user}, nil
}

func (s *serviceImpl) GetQuickUsers() ([]domain.User, error) {
	return s.repo.GetQuickUsers()
}

// Dashboard
func (s *serviceImpl) GetDashboard(userID uint) (*dto.DashboardResponse, error) {
	now := time.Now()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// 1. All-time cash balance & current month cashflow
	allIncome, _, allNet, err := s.repo.GetCashflowSummary(userID, 0, 0)
	if err != nil {
		return nil, err
	}

	mIncome, mExpense, mNet, err := s.repo.GetCashflowSummary(userID, currentMonth, currentYear)
	if err != nil {
		return nil, err
	}

	// 2. Unpaid invoices
	unpaidCount, unpaidSum, err := s.repo.GetUnpaidInvoicesCount(userID)
	if err != nil {
		return nil, err
	}

	// 3. Active tasks count
	var activeTasksCount int64
	s.repo.DB().Model(&domain.Task{}).Where("user_id = ? AND status != ?", userID, "paid").Count(&activeTasksCount)

	// 4. Top 2 agenda tasks
	var topTasks []domain.Task
	s.repo.DB().Where("user_id = ? AND status != ?", userID, "paid").
		Order("due_date asc, id desc").Limit(2).Find(&topTasks)

	// 5. Up to 2 pending invoices
	var pendingInvoices []domain.Invoice
	s.repo.DB().Where("user_id = ? AND status != ?", userID, "paid").
		Order("due_date asc, id desc").Limit(2).Find(&pendingInvoices)

	// BR-17: Hero Card Angka Utama = Total Pemasukan Kas + Nilai Invoice Menunggu Pembayaran
	estimatedTotal := allIncome + unpaidSum

	return &dto.DashboardResponse{
		EstimatedTotalValue: estimatedTotal,
		RealCashBalance:     allNet,
		ActiveTasksCount:    int(activeTasksCount),
		UnpaidInvoicesCount: int(unpaidCount),
		UnpaidInvoicesTotal: unpaidSum,
		CashflowThisMonth: dto.CashflowSum{
			TotalIncome:  mIncome,
			TotalExpense: mExpense,
			NetCashflow:  mNet,
		},
		TopAgendaTasks:  topTasks,
		PendingInvoices: pendingInvoices,
	}, nil
}

// Tasks
func (s *serviceImpl) GetTasks(userID uint, month, year int, taskType, status string, page, limit int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 4 // BR-01: Maksimal 4 per halaman
	}

	tasks, total, err := s.repo.GetTasks(userID, month, year, taskType, status, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return map[string]interface{}{
		"items": tasks,
		"pagination": dto.PageMeta{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *serviceImpl) GetTaskByID(userID, taskID uint) (*domain.Task, error) {
	return s.repo.GetTaskByID(userID, taskID)
}

func (s *serviceImpl) CreateTask(userID uint, req dto.TaskCreateRequest) (*domain.Task, error) {
	if req.Title == "" {
		return nil, errors.New("judul tugas wajib diisi")
	}

	brand := req.Brand
	if req.Type == "other" {
		brand = "Internal Studio" // BR-13
		req.FeeOrValue = 0
	} else if brand == "" {
		return nil, errors.New("nama brand wajib diisi untuk endorse & barter")
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		dueDate = time.Now().AddDate(0, 0, 7)
	}

	status := req.Status
	if status == "" {
		status = "draft"
	}

	var checklist []domain.ChecklistItem
	for _, itemText := range req.Checklist {
		if itemText != "" {
			checklist = append(checklist, domain.ChecklistItem{Text: itemText, Completed: false})
		}
	}

	task := &domain.Task{
		UserID:         userID,
		Type:           req.Type,
		Title:          req.Title,
		Brand:          brand,
		FeeOrValue:     req.FeeOrValue,
		DueDate:        dueDate,
		Deliverable:    req.Deliverable,
		Platform:       req.Platform,
		ShipmentNumber: req.ShipmentNumber,
		PicName:        req.PicName,
		PicPhone:       req.PicPhone,
		Priority:       req.Priority,
		Status:         status,
		Brief:          req.Brief,
		Checklist:      checklist,
	}

	if err := s.repo.CreateTask(task); err != nil {
		return nil, err
	}

	// Audit Log
	user, _ := s.repo.GetUserByID(userID)
	actor := "Creator"
	if user != nil {
		actor = user.Name
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     actor,
		Module:    "Tugas",
		Action:    "CREATE_TASK",
		Title:     fmt.Sprintf("Membuat Tugas Baru: %s", task.Title),
		Detail:    fmt.Sprintf("Tipe: %s, Brand: %s, Nilai: Rp %d", task.Type, task.Brand, task.FeeOrValue),
		CreatedAt: time.Now(),
	})

	return task, nil
}

func (s *serviceImpl) UpdateTaskStatus(userID, taskID uint, status string) (*domain.Task, error) {
	task, err := s.repo.GetTaskByID(userID, taskID)
	if err != nil {
		return nil, err
	}

	// Paid Lock Protection: cannot transition back once paid
	if task.Status == "paid" && status != "paid" {
		return nil, errors.New("tugas sudah lunas dan terkunci (paid lock)")
	}

	oldStatus := task.Status
	task.Status = status
	if err := s.repo.UpdateTask(task); err != nil {
		return nil, err
	}

	user, _ := s.repo.GetUserByID(userID)
	actor := "Creator"
	if user != nil {
		actor = user.Name
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     actor,
		Module:    "Tugas",
		Action:    "UPDATE_TASK_STATUS",
		Title:     fmt.Sprintf("Update Status Tugas: %s", task.Title),
		Detail:    fmt.Sprintf("Status berubah dari '%s' menjadi '%s'", oldStatus, status),
		CreatedAt: time.Now(),
	})

	return task, nil
}

func (s *serviceImpl) ToggleChecklistItem(userID, taskID, itemID uint) (*domain.ChecklistItem, error) {
	return s.repo.ToggleChecklistItem(userID, taskID, itemID)
}

func (s *serviceImpl) MarkTaskPaid(userID, taskID uint, req dto.PaymentConfirmationRequest) error {
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		// 1. Lock and fetch task
		var task domain.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
			return err
		}

		if task.Status == "paid" {
			return errors.New("tugas sudah lunas dan terkunci")
		}

		userSetting, err := s.repo.GetUserByID(userID)
		if err != nil {
			return err
		}

		// 2. Link or Auto-generate Invoice (BR-06)
		var invoice domain.Invoice
		if task.RelatedInvoiceID != nil {
			if err := tx.Where("id = ? AND user_id = ?", *task.RelatedInvoiceID, userID).First(&invoice).Error; err != nil {
				return err
			}
		} else {
			// Generate invoice number
			year := time.Now().Year()
			seq, err := s.repo.NextInvoiceSequence(tx, year, "ENDORSE")
			if err != nil {
				return err
			}
			invNumber := fmt.Sprintf("INV/ENDORSE/%d/%03d", year, seq)

			taxRate := userSetting.TaxRate
			subtotal := task.FeeOrValue
			taxAmount := (subtotal * int64(taxRate*100)) / 10000
			grandTotal := subtotal + taxAmount

			invoice = domain.Invoice{
				UserID:        userID,
				RelatedTaskID: &task.ID,
				InvoiceNumber: invNumber,
				ClientName:    task.Brand,
				IssueDate:     time.Now(),
				DueDate:       task.DueDate,
				IncludeTax:    true,
				TaxRate:       taxRate,
				Subtotal:      subtotal,
				TaxAmount:     taxAmount,
				GrandTotal:    grandTotal,
				Status:        "unpaid",
				Items: []domain.InvoiceItem{
					{Description: task.Title, Quantity: 1, UnitPrice: subtotal, Total: subtotal},
				},
			}
			if err := tx.Create(&invoice).Error; err != nil {
				return err
			}
			task.RelatedInvoiceID = &invoice.ID
		}

		// 3. Mark both as Paid with Proof
		task.Status = "paid"
		task.PaymentProofURL = req.PaymentProofURL
		task.PaymentProofName = req.PaymentProofName
		if err := tx.Save(&task).Error; err != nil {
			return err
		}

		invoice.Status = "paid"
		invoice.PaymentProofURL = req.PaymentProofURL
		invoice.PaymentProofName = req.PaymentProofName
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}

		// 4. Automatically Book Cashflow Income (BR-07, BR-08, BR-09)
		cashflow := domain.Transaction{
			UserID:           userID,
			RelatedInvoiceID: &invoice.ID,
			Type:             "income",
			Amount:           invoice.GrandTotal,
			Description:      fmt.Sprintf("Pelunasan Invoice %s (%s)", invoice.InvoiceNumber, task.Brand),
			Category:         "Pembayaran Endorse",
			Date:             time.Now(),
			AttachmentURL:    req.PaymentProofURL,
			AttachmentName:   req.PaymentProofName,
		}
		if err := tx.Create(&cashflow).Error; err != nil {
			return err
		}

		// 5. Audit Log (Lampiran A)
		audit := domain.AuditLog{
			UserID:    userID,
			Actor:     userSetting.Name,
			Module:    "Tugas",
			Action:    "MARK_PAID",
			Title:     fmt.Sprintf("Pelunasan Tugas %s", task.Title),
			Detail:    fmt.Sprintf("Pelunasan invoice %s senilai Rp %d dan pencatatan kas otomatis", invoice.InvoiceNumber, invoice.GrandTotal),
			CreatedAt: time.Now(),
		}
		return tx.Create(&audit).Error
	})
}

func (s *serviceImpl) DeleteTask(userID, taskID uint) error {
	task, err := s.repo.GetTaskByID(userID, taskID)
	if err != nil {
		return err
	}

	// Delete protection: Cannot delete if paid
	if task.Status == "paid" {
		return errors.New("tugas yang sudah lunas tidak dapat dihapus untuk melindungi integritas kas")
	}

	// If linked to an unpaid invoice, detach or delete draft invoice
	if task.RelatedInvoiceID != nil {
		var inv domain.Invoice
		if err := s.repo.DB().Where("id = ? AND user_id = ?", *task.RelatedInvoiceID, userID).First(&inv).Error; err == nil {
			if inv.Status != "paid" {
				_ = s.repo.DB().Delete(&inv)
			}
		}
	}

	if err := s.repo.DeleteTask(userID, taskID); err != nil {
		return err
	}

	user, _ := s.repo.GetUserByID(userID)
	actor := "Creator"
	if user != nil {
		actor = user.Name
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     actor,
		Module:    "Tugas",
		Action:    "DELETE_TASK",
		Title:     fmt.Sprintf("Menghapus Tugas: %s", task.Title),
		Detail:    fmt.Sprintf("Brand: %s", task.Brand),
		CreatedAt: time.Now(),
	})

	return nil
}

// Invoices
func (s *serviceImpl) GetInvoices(userID uint, month, year int, status string, page, limit int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 4 // BR-01
	}

	invoices, total, err := s.repo.GetInvoices(userID, month, year, status, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	unpaidCount, _, _ := s.repo.GetUnpaidInvoicesCount(userID)

	return map[string]interface{}{
		"items": invoices,
		"pagination": dto.PageMeta{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
		"unpaid_count": unpaidCount,
	}, nil
}

func (s *serviceImpl) GetInvoiceByID(userID, invoiceID uint) (*domain.Invoice, error) {
	return s.repo.GetInvoiceByID(userID, invoiceID)
}

func (s *serviceImpl) CreateInvoice(userID uint, req dto.InvoiceCreateRequest) (*domain.Invoice, error) {
	if req.ClientName == "" {
		return nil, errors.New("nama klien wajib diisi")
	}
	if len(req.Items) == 0 {
		return nil, errors.New("minimal satu baris rincian layanan wajib diisi")
	}

	userSetting, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		dueDate = time.Now().AddDate(0, 0, 14)
	}

	var subtotal int64
	var items []domain.InvoiceItem
	for _, it := range req.Items {
		itemTotal := int64(it.Quantity) * it.UnitPrice
		subtotal += itemTotal
		items = append(items, domain.InvoiceItem{
			Description: it.Description,
			Quantity:    it.Quantity,
			UnitPrice:   it.UnitPrice,
			Total:       itemTotal,
		})
	}

	taxRate := userSetting.TaxRate
	var taxAmount int64
	if req.IncludeTax {
		taxAmount = (subtotal * int64(taxRate*100)) / 10000
	} else {
		taxRate = 0
	}
	grandTotal := subtotal + taxAmount

	year := time.Now().Year()
	seq, err := s.repo.NextInvoiceSequence(s.repo.DB(), year, "ENDORSE")
	if err != nil {
		return nil, err
	}
	invNumber := fmt.Sprintf("INV/ENDORSE/%d/%03d", year, seq)

	invoice := &domain.Invoice{
		UserID:        userID,
		RelatedTaskID: req.RelatedTaskID,
		InvoiceNumber: invNumber,
		ClientName:    req.ClientName,
		ClientEmail:   req.ClientEmail,
		ClientPhone:   req.ClientPhone,
		ClientAddress: req.ClientAddress,
		IssueDate:     time.Now(),
		DueDate:       dueDate,
		IncludeTax:    req.IncludeTax,
		TaxRate:       taxRate,
		Subtotal:      subtotal,
		TaxAmount:     taxAmount,
		GrandTotal:    grandTotal,
		Status:        "unpaid",
		Items:         items,
	}

	if err := s.repo.CreateInvoice(invoice); err != nil {
		return nil, err
	}

	// Audit Log
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     userSetting.Name,
		Module:    "Invoice",
		Action:    "CREATE_INVOICE",
		Title:     fmt.Sprintf("Menerbitkan Invoice: %s", invoice.InvoiceNumber),
		Detail:    fmt.Sprintf("Klien: %s, Total: Rp %d", invoice.ClientName, invoice.GrandTotal),
		CreatedAt: time.Now(),
	})

	return invoice, nil
}

func (s *serviceImpl) ToggleInvoiceTax(userID, invoiceID uint) (*domain.Invoice, error) {
	inv, err := s.repo.GetInvoiceByID(userID, invoiceID)
	if err != nil {
		return nil, err
	}

	if inv.Status == "paid" {
		return nil, errors.New("pajak invoice yang sudah lunas tidak dapat diubah (tax lock)")
	}

	userSetting, _ := s.repo.GetUserByID(userID)
	inv.IncludeTax = !inv.IncludeTax
	if inv.IncludeTax {
		inv.TaxRate = userSetting.TaxRate
		inv.TaxAmount = (inv.Subtotal * int64(inv.TaxRate*100)) / 10000
	} else {
		inv.TaxRate = 0
		inv.TaxAmount = 0
	}
	inv.GrandTotal = inv.Subtotal + inv.TaxAmount

	if err := s.repo.UpdateInvoice(inv); err != nil {
		return nil, err
	}

	actor := "Creator"
	if userSetting != nil {
		actor = userSetting.Name
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     actor,
		Module:    "Invoice",
		Action:    "UPDATE_INVOICE_TAX",
		Title:     fmt.Sprintf("Toggle Pajak Invoice %s", inv.InvoiceNumber),
		Detail:    fmt.Sprintf("IncludeTax: %t, Baru: Rp %d", inv.IncludeTax, inv.GrandTotal),
		CreatedAt: time.Now(),
	})

	return inv, nil
}

func (s *serviceImpl) MarkInvoicePaid(userID, invoiceID uint, req dto.PaymentConfirmationRequest) error {
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		var inv domain.Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND user_id = ?", invoiceID, userID).First(&inv).Error; err != nil {
			return err
		}

		if inv.Status == "paid" {
			return errors.New("invoice sudah lunas dan terkunci")
		}

		inv.Status = "paid"
		inv.PaymentProofURL = req.PaymentProofURL
		inv.PaymentProofName = req.PaymentProofName
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}

		// Synchronize related task if exists
		if inv.RelatedTaskID != nil {
			var task domain.Task
			if err := tx.Where("id = ? AND user_id = ?", *inv.RelatedTaskID, userID).First(&task).Error; err == nil {
				task.Status = "paid"
				task.PaymentProofURL = req.PaymentProofURL
				task.PaymentProofName = req.PaymentProofName
				_ = tx.Save(&task)
			}
		}

		// Book cashflow entry (BR-07, BR-08, BR-09)
		cashflow := domain.Transaction{
			UserID:           userID,
			RelatedInvoiceID: &inv.ID,
			Type:             "income",
			Amount:           inv.GrandTotal,
			Description:      fmt.Sprintf("Pelunasan Tagihan %s (%s)", inv.InvoiceNumber, inv.ClientName),
			Category:         "Pembayaran Endorse",
			Date:             time.Now(),
			AttachmentURL:    req.PaymentProofURL,
			AttachmentName:   req.PaymentProofName,
		}
		if err := tx.Create(&cashflow).Error; err != nil {
			return err
		}

		userSetting, _ := s.repo.GetUserByID(userID)
		actor := "Creator"
		if userSetting != nil {
			actor = userSetting.Name
		}
		_ = tx.Create(&domain.AuditLog{
			UserID:    userID,
			Actor:     actor,
			Module:    "Invoice",
			Action:    "MARK_PAID",
			Title:     fmt.Sprintf("Pelunasan Invoice %s", inv.InvoiceNumber),
			Detail:    fmt.Sprintf("Total tagihan Rp %d telah dilunasi dan kas otomatis dibukukan", inv.GrandTotal),
			CreatedAt: time.Now(),
		})

		return nil
	})
}

// Cashflow
func (s *serviceImpl) GetCashflow(userID uint, month, year int, txType string, page, limit int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 4 // BR-01
	}

	txs, total, err := s.repo.GetTransactions(userID, month, year, txType, page, limit)
	if err != nil {
		return nil, err
	}

	income, expense, net, _ := s.repo.GetCashflowSummary(userID, month, year)

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return map[string]interface{}{
		"items": txs,
		"pagination": dto.PageMeta{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
		"summary": dto.CashflowSum{
			TotalIncome:  income,
			TotalExpense: expense,
			NetCashflow:  net,
		},
	}, nil
}

func (s *serviceImpl) CreateTransaction(userID uint, req dto.TransactionCreateRequest) (*domain.Transaction, error) {
	if req.Amount <= 0 {
		return nil, errors.New("nominal transaksi harus lebih dari 0")
	}
	if req.Description == "" {
		return nil, errors.New("deskripsi transaksi wajib diisi")
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		date = time.Now()
	}

	tx := &domain.Transaction{
		UserID:         userID,
		Type:           req.Type,
		Amount:         req.Amount,
		Description:    req.Description,
		Category:       req.Category,
		Date:           date,
		AttachmentURL:  req.AttachmentURL,
		AttachmentName: req.AttachmentName,
	}

	if err := s.repo.CreateTransaction(tx); err != nil {
		return nil, err
	}

	user, _ := s.repo.GetUserByID(userID)
	actor := "Creator"
	if user != nil {
		actor = user.Name
	}
	action := "CREATE_EXPENSE"
	if tx.Type == "income" {
		action = "CREATE_INCOME"
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     actor,
		Module:    "Keuangan",
		Action:    action,
		Title:     fmt.Sprintf("Pencatatan %s: %s", tx.Type, tx.Description),
		Detail:    fmt.Sprintf("Nominal: Rp %d, Kategori: %s", tx.Amount, tx.Category),
		CreatedAt: time.Now(),
	})

	return tx, nil
}

func (s *serviceImpl) DeleteTransaction(userID, txID uint) error {
	return s.repo.DeleteTransaction(userID, txID)
}

// Reports
func (s *serviceImpl) GetReports(userID uint, month, year int, coopType string, page, limit int) (*dto.ReportResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 4 // BR-01
	}

	monthNames := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}

	periodLabel := "Semua Periode"
	if year > 0 && month > 0 {
		periodLabel = fmt.Sprintf("%s %d", monthNames[month], year)
	} else if year > 0 {
		periodLabel = fmt.Sprintf("Tahun %d", year)
	}

	// Fetch all tasks for report according to period
	var tasks []domain.Task
	tq := s.repo.DB().Where("user_id = ?", userID)
	if year > 0 && month > 0 {
		startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(0, 1, 0)
		tq = tq.Where("due_date >= ? AND due_date < ?", startDate, endDate)
	} else if year > 0 {
		startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := startDate.AddDate(1, 0, 0)
		tq = tq.Where("due_date >= ? AND due_date < ?", startDate, endDate)
	}

	if coopType == "endorse" {
		tq = tq.Where("type = ?", "endorse")
	} else if coopType == "barter" {
		tq = tq.Where("type = ?", "barter")
	} else if coopType == "endorse+barter" {
		tq = tq.Where("type IN ?", []string{"endorse", "barter"})
	}

	tq.Order("due_date asc, id asc").Find(&tasks)

	// Combine with related invoices
	type ReportRow struct {
		Index            int    `json:"index"`
		TaskID           uint   `json:"task_id"`
		ProjectBrand     string `json:"project_brand"`
		Type             string `json:"type"`
		InvoiceNumber    string `json:"invoice_number"`
		Value            int64  `json:"value"`
		Status           string `json:"status"` // PAID / PENDING / NA
		PaymentProofURL  string `json:"payment_proof_url"`
		PaymentProofName string `json:"payment_proof_name"`
		AttachmentLabel  string `json:"attachment_label"` // #1, #2 or -
		DueDate          string `json:"due_date"`
	}

	var rows []ReportRow
	var totalFee int64
	var totalPaid int64
	var totalUnpaid int64
	var barterValuation int64
	attachmentIndex := 1

	for idx, t := range tasks {
		var invNumber = "-"
		val := t.FeeOrValue
		statusLabel := "PENDING"

		if t.Type == "barter" {
			statusLabel = "PENDING"
			if t.Status == "posting" {
				statusLabel = "SELESAI"
			}
			barterValuation += val
		} else if t.Type == "endorse" {
			if t.RelatedInvoiceID != nil {
				var inv domain.Invoice
				if err := s.repo.DB().First(&inv, *t.RelatedInvoiceID).Error; err == nil {
					invNumber = inv.InvoiceNumber
					val = inv.GrandTotal // BR-14: include PPN
					if inv.Status == "paid" {
						statusLabel = "LUNAS"
					}
				}
			}
			if t.Status == "paid" {
				statusLabel = "LUNAS"
			}

			totalFee += val
			if statusLabel == "LUNAS" {
				totalPaid += val
			} else {
				totalUnpaid += val
			}
		}

		attLabel := "-"
		if t.PaymentProofURL != "" {
			attLabel = fmt.Sprintf("#%d", attachmentIndex)
			attachmentIndex++
		}

		rows = append(rows, ReportRow{
			Index:            idx + 1,
			TaskID:           t.ID,
			ProjectBrand:     fmt.Sprintf("%s - %s", t.Brand, t.Title),
			Type:             t.Type,
			InvoiceNumber:    invNumber,
			Value:            val,
			Status:           statusLabel,
			PaymentProofURL:  t.PaymentProofURL,
			PaymentProofName: t.PaymentProofName,
			AttachmentLabel:  attLabel,
			DueDate:          t.DueDate.Format("2006-01-02"),
		})
	}

	totalItems := int64(len(rows))
	totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	start := (page - 1) * limit
	end := start + limit
	if start > len(rows) {
		start = len(rows)
	}
	if end > len(rows) {
		end = len(rows)
	}
	pagedRows := rows[start:end]

	return &dto.ReportResponse{
		PeriodLabel:     periodLabel,
		FilterType:      coopType,
		TotalProjects:   len(tasks),
		TotalFeeRevenue: totalFee,
		TotalPaid:       totalPaid,
		TotalUnpaid:     totalUnpaid,
		BarterValuation: barterValuation,
		Projects:        pagedRows,
		Pagination: dto.PageMeta{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  totalItems,
			TotalPages:  totalPages,
		},
	}, nil
}

// Settings
func (s *serviceImpl) GetSettings(userID uint) (*domain.User, error) {
	return s.repo.GetUserByID(userID)
}

func (s *serviceImpl) UpdateSettings(userID uint, req dto.SettingsUpdateRequest) (*domain.User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Company != "" {
		user.Company = req.Company
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.CompanyAddress != "" {
		user.CompanyAddress = req.CompanyAddress
	}
	if req.NPWP != "" {
		user.NPWP = req.NPWP
	}
	if req.TaxRate >= 0 && req.TaxRate <= 100 {
		user.TaxRate = req.TaxRate
	}
	user.LogoURL = req.LogoURL
	user.SignatureURL = req.SignatureURL
	if req.BankName != "" {
		user.BankName = req.BankName
	}
	if req.BankAccountNumber != "" {
		user.BankAccountNumber = req.BankAccountNumber
	}
	if req.BankAccountHolder != "" {
		user.BankAccountHolder = req.BankAccountHolder
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}

	if err := s.repo.UpdateUser(user); err != nil {
		return nil, err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    userID,
		Actor:     user.Name,
		Module:    "Setting",
		Action:    "UPDATE_SETTINGS",
		Title:     "Pembaruan Profil & Pengaturan Studio",
		Detail:    fmt.Sprintf("Perusahaan: %s, Tarif PPN Default: %.1f%%", user.Company, user.TaxRate),
		CreatedAt: time.Now(),
	})

	return user, nil
}

// Audit Logs
func (s *serviceImpl) GetAuditLogs(userID uint, module, search string, page, limit int) (map[string]interface{}, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}

	logs, total, err := s.repo.GetAuditLogs(userID, module, search, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	return map[string]interface{}{
		"items": logs,
		"pagination": dto.PageMeta{
			CurrentPage: page,
			PerPage:     limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func (s *serviceImpl) GetAvailableMonths(userID uint) ([]map[string]interface{}, error) {
	return s.repo.GetAvailableMonths(userID)
}
