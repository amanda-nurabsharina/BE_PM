package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
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
	ChangePassword(userID uint, req dto.ChangePasswordRequest) error

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

	// User Management (Super Admin only)
	GetUsers(requesterID uint) ([]dto.UserDetailResponse, error)
	CreateUser(requesterID uint, req dto.UserCreateRequest) (*dto.UserDetailResponse, error)
	UpdateUser(requesterID uint, targetUserID uint, req dto.UserUpdateRequest) (*dto.UserDetailResponse, error)
	DeleteUser(requesterID uint, targetUserID uint) error

	// Role Management
	GetRoles() ([]dto.RoleResponse, error)
	CreateRole(requesterID uint, req dto.RoleRequest) (*dto.RoleResponse, error)
	UpdateRole(requesterID uint, id uint, req dto.RoleRequest) (*dto.RoleResponse, error)
	DeleteRole(requesterID uint, id uint) error

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

func (s *serviceImpl) getOwnerID(userID uint) uint {
	user, err := s.repo.GetUserByID(userID)
	if err == nil && user != nil && user.OwnerID != nil && *user.OwnerID > 0 {
		return *user.OwnerID
	}
	return userID
}

func (s *serviceImpl) toUserDetail(user *domain.User) dto.UserDetailResponse {
	var menuList []string
	if user.Role == "SUPER_ADMIN" {
		menuList = []string{"beranda", "tugas", "laporan", "invoice", "keuangan", "pengaturan"}
	} else if user.AllowedMenus != "" {
		for _, m := range strings.Split(user.AllowedMenus, ",") {
			if t := strings.TrimSpace(m); t != "" {
				menuList = append(menuList, t)
			}
		}
	} else {
		if roleObj, err := s.repo.GetRoleByCode(user.Role); err == nil && roleObj != nil && roleObj.Menus != "" {
			for _, m := range strings.Split(roleObj.Menus, ",") {
				if t := strings.TrimSpace(m); t != "" {
					menuList = append(menuList, t)
				}
			}
		}
	}
	if len(menuList) == 0 {
		menuList = []string{"beranda"}
	}

	roleName := user.Role
	if user.Role == "SUPER_ADMIN" {
		roleName = "Super Admin"
	} else if r, err := s.repo.GetRoleByCode(user.Role); err == nil && r != nil {
		roleName = r.Name
	}

	return dto.UserDetailResponse{
		ID:           user.ID,
		Email:        user.Email,
		Name:         user.Name,
		AvatarURL:    user.AvatarURL,
		Role:         user.Role,
		RoleName:     roleName,
		AllowedMenus: menuList,
		OwnerID:      user.OwnerID,
		CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04"),
	}
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

	return &dto.LoginResponse{Token: token, User: s.toUserDetail(user)}, nil
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

	return &dto.LoginResponse{Token: token, User: s.toUserDetail(user)}, nil
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

	rows := make([]ReportRow, 0)
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
	pagedRows := make([]ReportRow, 0)
	if start < len(rows) {
		pagedRows = rows[start:end]
	}

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

// Change Password
func (s *serviceImpl) ChangePassword(userID uint, req dto.ChangePasswordRequest) error {
	if req.NewPassword == "" {
		return errors.New("password baru tidak boleh kosong")
	}
	if len(req.NewPassword) < 6 {
		return errors.New("password baru minimal 6 karakter")
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return errors.New("pengguna tidak ditemukan")
	}

	if req.OldPassword != "" && !security.ComparePassword(req.OldPassword, user.PasswordHash) {
		return errors.New("password lama tidak sesuai")
	}

	newHash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return errors.New("gagal mengenkripsi password")
	}

	if err := s.repo.UpdatePassword(userID, newHash); err != nil {
		return err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    s.getOwnerID(userID),
		Actor:     user.Name,
		Module:    "Setting",
		Action:    "CHANGE_PASSWORD",
		Title:     "Perubahan Password Akun",
		Detail:    fmt.Sprintf("Pengguna %s berhasil memperbarui password", user.Email),
		CreatedAt: time.Now(),
	})

	return nil
}

// User Management (Super Admin only)
func (s *serviceImpl) GetUsers(requesterID uint) ([]dto.UserDetailResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return nil, errors.New("hanya Super Admin yang memiliki hak akses mengelola pengguna")
	}

	ownerID := s.getOwnerID(requesterID)
	users, err := s.repo.GetUsers(ownerID)
	if err != nil {
		return nil, err
	}

	res := make([]dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		userCopy := u
		res = append(res, s.toUserDetail(&userCopy))
	}

	return res, nil
}

func (s *serviceImpl) CreateUser(requesterID uint, req dto.UserCreateRequest) (*dto.UserDetailResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return nil, errors.New("hanya Super Admin yang berwenang menambahkan pengguna baru")
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		return nil, errors.New("nama, email, dan password wajib diisi")
	}

	if len(req.Password) < 6 {
		return nil, errors.New("password minimal 6 karakter")
	}

	if existing, _ := s.repo.GetUserByEmail(req.Email); existing != nil && existing.ID > 0 {
		return nil, errors.New("email sudah terdaftar dalam sistem")
	}

	roleCode := req.Role
	if roleCode == "" {
		roleCode = "STAFF"
	}

	menusStr := strings.Join(req.AllowedMenus, ",")
	if roleCode == "SUPER_ADMIN" {
		menusStr = "beranda,tugas,laporan,invoice,keuangan,pengaturan"
	}

	ownerID := s.getOwnerID(requesterID)
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("gagal mengenkripsi password")
	}

	newUser := domain.User{
		Email:             req.Email,
		PasswordHash:      hashed,
		Name:              req.Name,
		AvatarURL:         req.AvatarURL,
		Role:              roleCode,
		AllowedMenus:      menusStr,
		OwnerID:           &ownerID,
		Company:           reqUser.Company,
		CompanyAddress:    reqUser.CompanyAddress,
		TaxRate:           reqUser.TaxRate,
		LogoURL:           reqUser.LogoURL,
		SignatureURL:      reqUser.SignatureURL,
		BankName:          reqUser.BankName,
		BankAccountNumber: reqUser.BankAccountNumber,
		BankAccountHolder: reqUser.BankAccountHolder,
	}

	if err := s.repo.CreateUser(&newUser); err != nil {
		return nil, err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    ownerID,
		Actor:     reqUser.Name,
		Module:    "Setting",
		Action:    "CREATE_USER",
		Title:     "Penambahan Pengguna Baru",
		Detail:    fmt.Sprintf("User baru %s (%s) dengan role %s berhasil didaftarkan", newUser.Name, newUser.Email, newUser.Role),
		CreatedAt: time.Now(),
	})

	detail := s.toUserDetail(&newUser)
	return &detail, nil
}

func (s *serviceImpl) UpdateUser(requesterID uint, targetUserID uint, req dto.UserUpdateRequest) (*dto.UserDetailResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return nil, errors.New("hanya Super Admin yang berwenang mengubah data pengguna")
	}

	target, err := s.repo.GetUserByID(targetUserID)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	if req.Name != "" {
		target.Name = req.Name
	}
	if req.Email != "" && req.Email != target.Email {
		if ex, _ := s.repo.GetUserByEmail(req.Email); ex != nil && ex.ID != target.ID {
			return nil, errors.New("email sudah digunakan oleh pengguna lain")
		}
		target.Email = req.Email
	}
	if req.AvatarURL != "" {
		target.AvatarURL = req.AvatarURL
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			return nil, errors.New("password baru minimal 6 karakter")
		}
		h, err := security.HashPassword(req.Password)
		if err == nil {
			target.PasswordHash = h
		}
	}

	// Super Admin primary owner cannot have role revoked
	if target.OwnerID == nil && target.Role == "SUPER_ADMIN" {
		target.Role = "SUPER_ADMIN"
		target.AllowedMenus = "beranda,tugas,laporan,invoice,keuangan,pengaturan"
	} else {
		if req.Role != "" {
			target.Role = req.Role
		}
		if len(req.AllowedMenus) > 0 {
			target.AllowedMenus = strings.Join(req.AllowedMenus, ",")
		}
	}

	if err := s.repo.UpdateUser(target); err != nil {
		return nil, err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    s.getOwnerID(requesterID),
		Actor:     reqUser.Name,
		Module:    "Setting",
		Action:    "UPDATE_USER",
		Title:     "Pembaruan Data Pengguna",
		Detail:    fmt.Sprintf("User %s diperbarui oleh %s", target.Email, reqUser.Name),
		CreatedAt: time.Now(),
	})

	detail := s.toUserDetail(target)
	return &detail, nil
}

func (s *serviceImpl) DeleteUser(requesterID uint, targetUserID uint) error {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return errors.New("hanya Super Admin yang berwenang menghapus pengguna")
	}

	if requesterID == targetUserID {
		return errors.New("tidak dapat menghapus akun Anda sendiri")
	}

	target, err := s.repo.GetUserByID(targetUserID)
	if err != nil {
		return errors.New("pengguna tidak ditemukan")
	}

	if target.OwnerID == nil && target.Role == "SUPER_ADMIN" {
		return errors.New("akun utama Super Admin tidak dapat dihapus")
	}

	if err := s.repo.DeleteUser(targetUserID); err != nil {
		return err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		UserID:    s.getOwnerID(requesterID),
		Actor:     reqUser.Name,
		Module:    "Setting",
		Action:    "DELETE_USER",
		Title:     "Penghapusan Pengguna",
		Detail:    fmt.Sprintf("User %s (%s) dihapus oleh %s", target.Name, target.Email, reqUser.Name),
		CreatedAt: time.Now(),
	})

	return nil
}

// Role Management
func (s *serviceImpl) GetRoles() ([]dto.RoleResponse, error) {
	roles, err := s.repo.GetRoles()
	if err != nil {
		return nil, err
	}

	res := make([]dto.RoleResponse, 0, len(roles))
	for _, r := range roles {
		var menuList []string
		if r.Menus != "" {
			for _, m := range strings.Split(r.Menus, ",") {
				if t := strings.TrimSpace(m); t != "" {
					menuList = append(menuList, t)
				}
			}
		}
		res = append(res, dto.RoleResponse{
			ID:          r.ID,
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
			Menus:       menuList,
			IsSystem:    r.IsSystem,
		})
	}
	return res, nil
}

func (s *serviceImpl) CreateRole(requesterID uint, req dto.RoleRequest) (*dto.RoleResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return nil, errors.New("hanya Super Admin yang berwenang membuat role baru")
	}

	if req.Code == "" || req.Name == "" {
		return nil, errors.New("kode dan nama role wajib diisi")
	}

	cleanCode := strings.ToUpper(strings.TrimSpace(req.Code))
	if ex, _ := s.repo.GetRoleByCode(cleanCode); ex != nil && ex.ID > 0 {
		return nil, errors.New("kode role sudah ada")
	}

	menusStr := strings.Join(req.Menus, ",")
	newRole := domain.Role{
		Code:        cleanCode,
		Name:        req.Name,
		Description: req.Description,
		Menus:       menusStr,
		IsSystem:    false,
	}

	if err := s.repo.CreateRole(&newRole); err != nil {
		return nil, err
	}

	return &dto.RoleResponse{
		ID:          newRole.ID,
		Code:        newRole.Code,
		Name:        newRole.Name,
		Description: newRole.Description,
		Menus:       req.Menus,
		IsSystem:    false,
	}, nil
}

func (s *serviceImpl) UpdateRole(requesterID uint, id uint, req dto.RoleRequest) (*dto.RoleResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return nil, errors.New("hanya Super Admin yang berwenang mengubah role")
	}

	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return nil, errors.New("role tidak ditemukan")
	}

	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if len(req.Menus) > 0 {
		if role.Code == "SUPER_ADMIN" {
			role.Menus = "beranda,tugas,laporan,invoice,keuangan,pengaturan"
		} else {
			role.Menus = strings.Join(req.Menus, ",")
		}
	}

	if err := s.repo.UpdateRole(role); err != nil {
		return nil, err
	}

	var menuList []string
	for _, m := range strings.Split(role.Menus, ",") {
		if t := strings.TrimSpace(m); t != "" {
			menuList = append(menuList, t)
		}
	}

	return &dto.RoleResponse{
		ID:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Menus:       menuList,
		IsSystem:    role.IsSystem,
	}, nil
}

func (s *serviceImpl) DeleteRole(requesterID uint, id uint) error {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || reqUser.Role != "SUPER_ADMIN" {
		return errors.New("hanya Super Admin yang berwenang menghapus role")
	}

	role, err := s.repo.GetRoleByID(id)
	if err != nil {
		return errors.New("role tidak ditemukan")
	}

	if role.IsSystem || role.Code == "SUPER_ADMIN" {
		return errors.New("role sistem tidak dapat dihapus")
	}

	return s.repo.DeleteRole(id)
}
