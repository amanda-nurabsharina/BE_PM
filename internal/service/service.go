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
	AdminLogin(req dto.LoginRequest) (*dto.LoginResponse, error)
	QuickLogin(email string) (*dto.LoginResponse, error)
	GetQuickUsers() ([]domain.User, error)
	ChangePassword(userID uint, req dto.ChangePasswordRequest) error

	// Platform & Tenant Management
	GetTenants() ([]dto.TenantResponse, error)
	GetTenantByID(id uint) (*dto.TenantResponse, error)
	CreateTenant(req dto.CreateTenantRequest) (*dto.TenantResponse, error)
	UpdateTenantStatus(id uint, req dto.UpdateTenantStatusRequest) (*dto.TenantResponse, error)
	ResetTenantOwnerPassword(id uint, req dto.ResetTenantOwnerPasswordRequest) error

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

	// User Management (Super Admin of tenant)
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

func (s *serviceImpl) getTenantID(userID uint) uint {
	user, err := s.repo.GetUserByID(userID)
	if err == nil && user != nil && user.TenantID > 0 {
		return user.TenantID
	}
	return 1
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
	if user.Role == "SUPER_ADMIN" || user.Role == "OWNER" || user.IsOwner {
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
	} else if user.Role == "OWNER" || user.IsOwner {
		roleName = "Owner Studio"
	} else if r, err := s.repo.GetRoleByCode(user.Role); err == nil && r != nil {
		roleName = r.Name
	}

	var tenantResp *dto.TenantResponse
	tenantName := ""
	tenantSlug := ""
	if user.TenantID > 0 {
		tenant, err := s.repo.GetTenantByID(user.TenantID)
		if err == nil && tenant != nil {
			tenantName = tenant.Name
			tenantSlug = tenant.Slug
			userCount, taskCount, _ := s.repo.GetTenantStats(tenant.ID)
			tenantResp = &dto.TenantResponse{
				ID:                tenant.ID,
				Name:              tenant.Name,
				Slug:              tenant.Slug,
				Status:            tenant.Status,
				CompanyAddress:    tenant.CompanyAddress,
				Phone:             tenant.Phone,
				NPWP:              tenant.NPWP,
				TaxRate:           tenant.TaxRate,
				BankName:          tenant.BankName,
				BankAccountNumber: tenant.BankAccountNumber,
				BankAccountHolder: tenant.BankAccountHolder,
				OwnerID:           user.ID,
				OwnerName:         user.Name,
				OwnerEmail:        user.Email,
				TotalUsers:        userCount,
				TotalTasks:        taskCount,
				CreatedAt:         tenant.CreatedAt.Format("2006-01-02 15:04"),
			}
		}
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
		TenantID:     user.TenantID,
		TenantSlug:   tenantSlug,
		TenantName:   tenantName,
		IsOwner:      user.IsOwner,
		Tenant:       tenantResp,
		CreatedAt:    user.CreatedAt.Format("2006-01-02 15:04"),
	}
}

// Auth
func (s *serviceImpl) Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email dan password wajib diisi")
	}

	var user *domain.User
	var err error

	// If tenant slug is provided, search specifically in that tenant
	if req.TenantSlug != "" {
		tenant, tErr := s.repo.GetTenantBySlug(req.TenantSlug)
		if tErr != nil || tenant == nil {
			return nil, errors.New("tenant/workspace tidak ditemukan")
		}
		if tenant.Status == "suspended" {
			return nil, errors.New("tenant/workspace ini dinonaktifkan (suspended). Hubungi administrator")
		}
		user, err = s.repo.GetUserByEmailAndTenant(req.Email, tenant.ID)
	} else {
		user, err = s.repo.GetUserByEmail(req.Email)
	}

	if err != nil || user == nil {
		return nil, errors.New("kredensial login tidak valid")
	}

	if !security.ComparePassword(req.Password, user.PasswordHash) {
		return nil, errors.New("kredensial login tidak valid")
	}

	// Super Admin cannot login via Creator Task workspace portal
	if user.Role == "SUPER_ADMIN" {
		return nil, errors.New("Akun Super Admin adalah akun pengelola platform SaaS. Silakan login melalui portal Master Admin (/admin/login)")
	}

	tenantSlug := ""
	if user.TenantID > 0 {
		tenant, tErr := s.repo.GetTenantByID(user.TenantID)
		if tErr == nil && tenant != nil {
			tenantSlug = tenant.Slug
			if tenant.Status == "suspended" {
				return nil, errors.New("tenant ini sedang dinonaktifkan (suspended). Hubungi platform administrator")
			}
		}
	}

	token, err := security.GenerateToken(user.ID, user.TenantID, tenantSlug, user.Email, user.Name, user.Role, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{Token: token, User: s.toUserDetail(user)}, nil
}

func (s *serviceImpl) AdminLogin(req dto.LoginRequest) (*dto.LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email dan password wajib diisi")
	}

	user, err := s.repo.GetUserByEmail(req.Email)
	if err != nil || user == nil {
		return nil, errors.New("kredensial login tidak valid")
	}

	if !security.ComparePassword(req.Password, user.PasswordHash) {
		return nil, errors.New("kredensial login tidak valid")
	}

	if user.Role != "SUPER_ADMIN" {
		return nil, errors.New("Akses ditolak: Hanya akun Master Platform Super Admin yang dapat login ke portal ini")
	}

	token, err := security.GenerateToken(user.ID, 0, "platform-master", user.Email, user.Name, user.Role, s.cfg.JWTSecret, 24*time.Hour)
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

	tenantSlug := ""
	if user.TenantID > 0 {
		tenant, tErr := s.repo.GetTenantByID(user.TenantID)
		if tErr == nil && tenant != nil {
			tenantSlug = tenant.Slug
		}
	}

	token, err := security.GenerateToken(user.ID, user.TenantID, tenantSlug, user.Email, user.Name, user.Role, s.cfg.JWTSecret, 24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{Token: token, User: s.toUserDetail(user)}, nil
}

func (s *serviceImpl) GetQuickUsers() ([]domain.User, error) {
	return s.repo.GetQuickUsers()
}

// Platform Tenant Management
func (s *serviceImpl) GetTenants() ([]dto.TenantResponse, error) {
	tenants, err := s.repo.GetTenants()
	if err != nil {
		return nil, err
	}

	res := make([]dto.TenantResponse, 0, len(tenants))
	for _, t := range tenants {
		var owner domain.User
		s.repo.DB().Where("tenant_id = ? AND (is_owner = ? OR role = ?)", t.ID, true, "SUPER_ADMIN").Order("id asc").First(&owner)
		userCount, taskCount, _ := s.repo.GetTenantStats(t.ID)

		res = append(res, dto.TenantResponse{
			ID:                t.ID,
			Name:              t.Name,
			Slug:              t.Slug,
			Status:            t.Status,
			CompanyAddress:    t.CompanyAddress,
			Phone:             t.Phone,
			NPWP:              t.NPWP,
			TaxRate:           t.TaxRate,
			BankName:          t.BankName,
			BankAccountNumber: t.BankAccountNumber,
			BankAccountHolder: t.BankAccountHolder,
			OwnerID:           owner.ID,
			OwnerName:         owner.Name,
			OwnerEmail:        owner.Email,
			TotalUsers:        userCount,
			TotalTasks:        taskCount,
			CreatedAt:         t.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	return res, nil
}

func (s *serviceImpl) GetTenantByID(id uint) (*dto.TenantResponse, error) {
	t, err := s.repo.GetTenantByID(id)
	if err != nil {
		return nil, errors.New("tenant tidak ditemukan")
	}

	var owner domain.User
	s.repo.DB().Where("tenant_id = ? AND (is_owner = ? OR role = ?)", t.ID, true, "SUPER_ADMIN").Order("id asc").First(&owner)
	userCount, taskCount, _ := s.repo.GetTenantStats(t.ID)

	return &dto.TenantResponse{
		ID:                t.ID,
		Name:              t.Name,
		Slug:              t.Slug,
		Status:            t.Status,
		CompanyAddress:    t.CompanyAddress,
		Phone:             t.Phone,
		NPWP:              t.NPWP,
		TaxRate:           t.TaxRate,
		BankName:          t.BankName,
		BankAccountNumber: t.BankAccountNumber,
		BankAccountHolder: t.BankAccountHolder,
		OwnerID:           owner.ID,
		OwnerName:         owner.Name,
		OwnerEmail:        owner.Email,
		TotalUsers:        userCount,
		TotalTasks:        taskCount,
		CreatedAt:         t.CreatedAt.Format("2006-01-02 15:04"),
	}, nil
}

func (s *serviceImpl) CreateTenant(req dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	if req.Name == "" || req.Slug == "" {
		return nil, errors.New("nama studio dan slug wajib diisi")
	}
	if req.OwnerName == "" || req.OwnerEmail == "" || req.OwnerPassword == "" {
		return nil, errors.New("nama owner, email, dan password wajib diisi")
	}
	if len(req.OwnerPassword) < 6 {
		return nil, errors.New("password owner minimal 6 karakter")
	}

	cleanSlug := strings.ToLower(strings.TrimSpace(req.Slug))
	cleanSlug = strings.ReplaceAll(cleanSlug, " ", "-")

	if existing, _ := s.repo.GetTenantBySlug(cleanSlug); existing != nil && existing.ID > 0 {
		return nil, errors.New("slug subdomain/id sudah digunakan")
	}

	if existingUser, _ := s.repo.GetUserByEmail(req.OwnerEmail); existingUser != nil && existingUser.ID > 0 {
		return nil, errors.New("email owner sudah terdaftar di sistem")
	}

	hashedPassword, err := security.HashPassword(req.OwnerPassword)
	if err != nil {
		return nil, errors.New("gagal mengenkripsi password")
	}

	taxRate := req.TaxRate
	if taxRate <= 0 {
		taxRate = 11.0
	}

	tenant := domain.Tenant{
		Name:              req.Name,
		Slug:              cleanSlug,
		Status:            "active",
		CompanyAddress:    req.CompanyAddress,
		Phone:             req.Phone,
		NPWP:              req.NPWP,
		TaxRate:           taxRate,
		BankName:          req.BankName,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountHolder: req.BankAccountHolder,
	}

	err = s.repo.DB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}

		owner := domain.User{
			TenantID:          tenant.ID,
			IsOwner:           true,
			Email:             req.OwnerEmail,
			PasswordHash:      hashedPassword,
			Name:              req.OwnerName,
			Role:              "OWNER",
			AllowedMenus:      "beranda,tugas,laporan,invoice,keuangan,pengaturan",
			Company:           req.Name,
			CompanyAddress:    req.CompanyAddress,
			Phone:             req.Phone,
			NPWP:              req.NPWP,
			TaxRate:           taxRate,
			BankName:          req.BankName,
			BankAccountNumber: req.BankAccountNumber,
			BankAccountHolder: req.BankAccountHolder,
		}

		if err := tx.Create(&owner).Error; err != nil {
			return err
		}

		tx.Create(&domain.AuditLog{
			TenantID:  tenant.ID,
			UserID:    owner.ID,
			Actor:     "Platform Admin",
			Module:    "Tenant",
			Action:    "CREATE_TENANT",
			Title:     fmt.Sprintf("Inisialisasi Tenant Baru: %s", tenant.Name),
			Detail:    fmt.Sprintf("Owner: %s (%s), Slug: %s", owner.Name, owner.Email, tenant.Slug),
			CreatedAt: time.Now(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetTenantByID(tenant.ID)
}

func (s *serviceImpl) UpdateTenantStatus(id uint, req dto.UpdateTenantStatusRequest) (*dto.TenantResponse, error) {
	if req.Status != "active" && req.Status != "suspended" {
		return nil, errors.New("status tidak valid (harus 'active' atau 'suspended')")
	}

	if err := s.repo.UpdateTenantStatus(id, req.Status); err != nil {
		return nil, err
	}

	return s.GetTenantByID(id)
}

func (s *serviceImpl) ResetTenantOwnerPassword(id uint, req dto.ResetTenantOwnerPasswordRequest) error {
	if len(req.NewPassword) < 6 {
		return errors.New("password baru minimal 6 karakter")
	}

	var owner domain.User
	if err := s.repo.DB().Where("tenant_id = ? AND (is_owner = ? OR role = ?)", id, true, "SUPER_ADMIN").Order("id asc").First(&owner).Error; err != nil {
		return errors.New("owner untuk tenant ini tidak ditemukan")
	}

	newHash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return errors.New("gagal mengenkripsi password")
	}

	return s.repo.UpdatePassword(owner.ID, newHash)
}

// Dashboard
func (s *serviceImpl) GetDashboard(userID uint) (*dto.DashboardResponse, error) {
	now := time.Now()
	currentMonth := int(now.Month())
	currentYear := now.Year()
	tenantID := s.getTenantID(userID)

	// 1. All-time cash balance & current month cashflow
	allIncome, _, allNet, err := s.repo.GetCashflowSummary(tenantID, 0, 0)
	if err != nil {
		return nil, err
	}

	mIncome, mExpense, mNet, err := s.repo.GetCashflowSummary(tenantID, currentMonth, currentYear)
	if err != nil {
		return nil, err
	}

	// 2. Unpaid invoices
	unpaidCount, unpaidSum, err := s.repo.GetUnpaidInvoicesCount(tenantID)
	if err != nil {
		return nil, err
	}

	// 3. Active tasks count
	var activeTasksCount int64
	s.repo.DB().Model(&domain.Task{}).Where("tenant_id = ? AND status != ?", tenantID, "paid").Count(&activeTasksCount)

	// 4. Top 2 agenda tasks
	var topTasks []domain.Task
	s.repo.DB().Where("tenant_id = ? AND status != ?", tenantID, "paid").
		Order("due_date asc, id desc").Limit(2).Find(&topTasks)

	// 5. Up to 2 pending invoices
	var pendingInvoices []domain.Invoice
	s.repo.DB().Where("tenant_id = ? AND status != ?", tenantID, "paid").
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
	tenantID := s.getTenantID(userID)

	tasks, total, err := s.repo.GetTasks(tenantID, month, year, taskType, status, page, limit)
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
	tenantID := s.getTenantID(userID)
	return s.repo.GetTaskByID(tenantID, taskID)
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

	tenantID := s.getTenantID(userID)
	task := &domain.Task{
		TenantID:       tenantID,
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
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)
	task, err := s.repo.GetTaskByID(tenantID, taskID)
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
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)
	return s.repo.ToggleChecklistItem(tenantID, taskID, itemID)
}

func (s *serviceImpl) MarkTaskPaid(userID, taskID uint, req dto.PaymentConfirmationRequest) error {
	tenantID := s.getTenantID(userID)
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		// 1. Lock and fetch task
		var task domain.Task
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND tenant_id = ?", taskID, tenantID).First(&task).Error; err != nil {
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
			if err := tx.Where("id = ? AND tenant_id = ?", *task.RelatedInvoiceID, tenantID).First(&invoice).Error; err != nil {
				return err
			}
		} else {
			// Generate invoice number
			year := time.Now().Year()
			seq, err := s.repo.NextInvoiceSequence(tx, tenantID, year, "ENDORSE")
			if err != nil {
				return err
			}
			invNumber := fmt.Sprintf("INV/ENDORSE/%d/%03d", year, seq)

			taxRate := userSetting.TaxRate
			subtotal := task.FeeOrValue
			taxAmount := (subtotal * int64(taxRate*100)) / 10000
			grandTotal := subtotal + taxAmount

			invoice = domain.Invoice{
				TenantID:      tenantID,
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
			TenantID:         tenantID,
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
			TenantID:  tenantID,
			UserID:    userID,
			Actor:     userSetting.Name,
			Module:    "Tugas",
			Action:    "MARK_PAID",
			Title:     fmt.Sprintf("Pelunasan Tugas %s", task.Title),
			Detail:    fmt.Sprintf("Tugas senilai Rp %d telah dilunasi beserta invoice %s", task.FeeOrValue, invoice.InvoiceNumber),
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *serviceImpl) DeleteTask(userID, taskID uint) error {
	tenantID := s.getTenantID(userID)
	task, err := s.repo.GetTaskByID(tenantID, taskID)
	if err != nil {
		return err
	}

	if task.Status == "paid" {
		return errors.New("tugas yang sudah lunas tidak dapat dihapus")
	}

	// Delete related draft/unpaid invoice if created automatically
	if task.RelatedInvoiceID != nil {
		var inv domain.Invoice
		if err := s.repo.DB().Where("id = ? AND tenant_id = ?", *task.RelatedInvoiceID, tenantID).First(&inv).Error; err == nil {
			if inv.Status != "paid" {
				_ = s.repo.DB().Delete(&inv)
			}
		}
	}

	if err := s.repo.DeleteTask(tenantID, taskID); err != nil {
		return err
	}

	user, _ := s.repo.GetUserByID(userID)
	actor := "Creator"
	if user != nil {
		actor = user.Name
	}
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)

	invoices, total, err := s.repo.GetInvoices(tenantID, month, year, status, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	unpaidCount, _, _ := s.repo.GetUnpaidInvoicesCount(tenantID)

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
	tenantID := s.getTenantID(userID)
	return s.repo.GetInvoiceByID(tenantID, invoiceID)
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

	tenantID := s.getTenantID(userID)
	year := time.Now().Year()
	seq, err := s.repo.NextInvoiceSequence(s.repo.DB(), tenantID, year, "ENDORSE")
	if err != nil {
		return nil, err
	}
	invNumber := fmt.Sprintf("INV/ENDORSE/%d/%03d", year, seq)

	invoice := &domain.Invoice{
		TenantID:      tenantID,
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
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)
	inv, err := s.repo.GetInvoiceByID(tenantID, invoiceID)
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
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)
	return s.repo.DB().Transaction(func(tx *gorm.DB) error {
		var inv domain.Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND tenant_id = ?", invoiceID, tenantID).First(&inv).Error; err != nil {
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
			if err := tx.Where("id = ? AND tenant_id = ?", *inv.RelatedTaskID, tenantID).First(&task).Error; err == nil {
				task.Status = "paid"
				task.PaymentProofURL = req.PaymentProofURL
				task.PaymentProofName = req.PaymentProofName
				_ = tx.Save(&task)
			}
		}

		// Book cashflow entry (BR-07, BR-08, BR-09)
		cashflow := domain.Transaction{
			TenantID:         tenantID,
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
			TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)

	txs, total, err := s.repo.GetTransactions(tenantID, month, year, txType, page, limit)
	if err != nil {
		return nil, err
	}

	income, expense, net, _ := s.repo.GetCashflowSummary(tenantID, month, year)

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

	tenantID := s.getTenantID(userID)
	tx := &domain.Transaction{
		TenantID:       tenantID,
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
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)
	return s.repo.DeleteTransaction(tenantID, txID)
}

// Reports
func (s *serviceImpl) GetReports(userID uint, month, year int, coopType string, page, limit int) (*dto.ReportResponse, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 4 // BR-01
	}
	tenantID := s.getTenantID(userID)

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

	// Fetch all tasks for report according to period and tenant
	var tasks []domain.Task
	tq := s.repo.DB().Where("tenant_id = ?", tenantID)
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
				if err := s.repo.DB().Where("id = ? AND tenant_id = ?", *t.RelatedInvoiceID, tenantID).First(&inv).Error; err == nil {
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

	tenantID := s.getTenantID(userID)
	// Also sync tenant company/tax info if this user is the owner
	if user.IsOwner && user.TenantID > 0 {
		tenant, tErr := s.repo.GetTenantByID(user.TenantID)
		if tErr == nil && tenant != nil {
			if req.Company != "" {
				tenant.Name = req.Company
			}
			if req.CompanyAddress != "" {
				tenant.CompanyAddress = req.CompanyAddress
			}
			if req.Phone != "" {
				tenant.Phone = req.Phone
			}
			if req.NPWP != "" {
				tenant.NPWP = req.NPWP
			}
			if req.TaxRate >= 0 && req.TaxRate <= 100 {
				tenant.TaxRate = req.TaxRate
			}
			if req.BankName != "" {
				tenant.BankName = req.BankName
			}
			if req.BankAccountNumber != "" {
				tenant.BankAccountNumber = req.BankAccountNumber
			}
			if req.BankAccountHolder != "" {
				tenant.BankAccountHolder = req.BankAccountHolder
			}
			_ = s.repo.UpdateTenant(tenant)
		}
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		TenantID:  tenantID,
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
	tenantID := s.getTenantID(userID)

	logs, total, err := s.repo.GetAuditLogs(tenantID, module, search, page, limit)
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
	tenantID := s.getTenantID(userID)
	return s.repo.GetAvailableMonths(tenantID)
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

	tenantID := s.getTenantID(userID)
	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		TenantID:  tenantID,
		UserID:    userID,
		Actor:     user.Name,
		Module:    "Setting",
		Action:    "CHANGE_PASSWORD",
		Title:     "Perubahan Password Akun",
		Detail:    fmt.Sprintf("Pengguna %s berhasil memperbarui password", user.Email),
		CreatedAt: time.Now(),
	})

	return nil
}

// User Management (Owner of tenant or Super Admin)
func (s *serviceImpl) GetUsers(requesterID uint) ([]dto.UserDetailResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || (!reqUser.IsOwner && reqUser.Role != "OWNER" && reqUser.Role != "SUPER_ADMIN") {
		return nil, errors.New("hanya Owner Studio atau Super Admin yang memiliki hak akses mengelola pengguna")
	}

	tenantID := s.getTenantID(requesterID)
	users, err := s.repo.GetUsers(tenantID)
	if err != nil {
		return nil, err
	}

	res := make([]dto.UserDetailResponse, 0, len(users))
	for _, u := range users {
		// Do not expose platform Super Admin in tenant's user management list
		if u.Email == "superadmin@creatortask.id" || u.Role == "SUPER_ADMIN" {
			continue
		}
		userCopy := u
		res = append(res, s.toUserDetail(&userCopy))
	}

	return res, nil
}

func (s *serviceImpl) CreateUser(requesterID uint, req dto.UserCreateRequest) (*dto.UserDetailResponse, error) {
	reqUser, err := s.repo.GetUserByID(requesterID)
	if err != nil || (!reqUser.IsOwner && reqUser.Role != "OWNER" && reqUser.Role != "SUPER_ADMIN") {
		return nil, errors.New("hanya Owner Studio atau Super Admin yang berwenang menambahkan pengguna baru")
	}

	if req.Role == "SUPER_ADMIN" {
		return nil, errors.New("role Super Admin hanya khusus untuk Platform Admin")
	}

	if req.Email == "" || req.Name == "" || req.Password == "" {
		return nil, errors.New("nama, email, dan password wajib diisi")
	}

	if len(req.Password) < 6 {
		return nil, errors.New("password minimal 6 karakter")
	}

	tenantID := s.getTenantID(requesterID)
	if existing, _ := s.repo.GetUserByEmailAndTenant(req.Email, tenantID); existing != nil && existing.ID > 0 {
		return nil, errors.New("email sudah terdaftar dalam workspace ini")
	}

	roleCode := req.Role
	if roleCode == "" {
		roleCode = "STAFF"
	}

	menusStr := strings.Join(req.AllowedMenus, ",")
	if roleCode == "SUPER_ADMIN" || roleCode == "OWNER" {
		menusStr = "beranda,tugas,laporan,invoice,keuangan,pengaturan"
	}

	ownerID := s.getOwnerID(requesterID)
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("gagal mengenkripsi password")
	}

	newUser := domain.User{
		TenantID:          tenantID,
		IsOwner:           false,
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
		TenantID:  tenantID,
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
	if err != nil || (!reqUser.IsOwner && reqUser.Role != "OWNER" && reqUser.Role != "SUPER_ADMIN") {
		return nil, errors.New("hanya Owner Studio atau Super Admin yang berwenang mengubah data pengguna")
	}

	if req.Role == "SUPER_ADMIN" {
		return nil, errors.New("role Super Admin hanya khusus untuk Platform Admin")
	}

	target, err := s.repo.GetUserByID(targetUserID)
	if err != nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	tenantID := s.getTenantID(requesterID)
	if target.TenantID != tenantID {
		return nil, errors.New("akses ditolak: pengguna tidak berada di workspace Anda")
	}

	if req.Name != "" {
		target.Name = req.Name
	}
	if req.Email != "" && req.Email != target.Email {
		if ex, _ := s.repo.GetUserByEmailAndTenant(req.Email, tenantID); ex != nil && ex.ID != target.ID {
			return nil, errors.New("email sudah digunakan oleh pengguna lain di workspace ini")
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

	// Owner primary user cannot have role revoked
	if target.IsOwner || target.Role == "OWNER" {
		target.Role = "OWNER"
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
		TenantID:  tenantID,
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
	if err != nil || (!reqUser.IsOwner && reqUser.Role != "OWNER" && reqUser.Role != "SUPER_ADMIN") {
		return errors.New("hanya Owner Studio atau Super Admin yang berwenang menghapus pengguna")
	}

	if requesterID == targetUserID {
		return errors.New("tidak dapat menghapus akun Anda sendiri")
	}

	target, err := s.repo.GetUserByID(targetUserID)
	if err != nil {
		return errors.New("pengguna tidak ditemukan")
	}

	tenantID := s.getTenantID(requesterID)
	if target.TenantID != tenantID {
		return errors.New("akses ditolak: pengguna tidak berada di workspace Anda")
	}

	if target.IsOwner || target.Role == "OWNER" {
		return errors.New("akun utama Owner Studio tidak dapat dihapus")
	}

	if err := s.repo.DeleteUser(targetUserID); err != nil {
		return err
	}

	_ = s.repo.CreateAuditLog(&domain.AuditLog{
		TenantID:  tenantID,
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
