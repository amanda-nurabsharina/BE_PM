package main

import (
	"fmt"
	"log"

	"be-pm/config"
	"be-pm/internal/handler"
	"be-pm/internal/middleware"
	"be-pm/internal/repository"
	"be-pm/internal/service"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()

	// Connect Database (PostgreSQL preferred, fallback to SQLite)
	var db *gorm.DB
	var err error

	if cfg.DBDriver == "postgres" {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode)
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Printf("Warning: Failed to connect to PostgreSQL (%v). Falling back to SQLite (%s)...", err, cfg.SQLitePath)
			db, err = gorm.Open(sqlite.Open(cfg.SQLitePath), &gorm.Config{})
		} else {
			log.Printf("Successfully connected to PostgreSQL database [%s] at %s:%s", cfg.DBName, cfg.DBHost, cfg.DBPort)
		}
	} else {
		db, err = gorm.Open(sqlite.Open(cfg.SQLitePath), &gorm.Config{})
	}

	if err != nil {
		log.Fatalf("Fatal: Failed to connect to any database: %v", err)
	}

	// Initialize Repository & Auto Migration
	repo := repository.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database tables migration completed successfully")

	// Seed Initial Setup (Roles and initial super admin account if needed)
	if err := repo.SeedInitialData(); err != nil {
		log.Printf("Warning during initial seeding: %v", err)
	} else {
		log.Println("Initial setup (roles & super admin) synchronized")
	}

	// Purge sample/mock data (tasks, invoices, cashflow, audit logs) while preserving login accounts & roles
	if err := repo.CleanSampleData(); err != nil {
		log.Printf("Warning during sample data cleanup: %v", err)
	}

	// Initialize Service Layer & Handlers
	svc := service.NewService(repo, cfg)
	authHandler := handler.NewAuthHandler(svc)
	taskHandler := handler.NewTaskHandler(svc)
	invoiceHandler := handler.NewInvoiceHandler(svc)
	cashflowHandler := handler.NewCashflowHandler(svc)
	reportHandler := handler.NewReportHandler(svc)
	settingHandler := handler.NewSettingHandler(svc)
	auditHandler := handler.NewAuditHandler(svc)
	uploadHandler := handler.NewUploadHandler(cfg.UploadDir)

	// Fiber Engine Setup with 2MB Body Limit for Upload Protection
	app := fiber.New(fiber.Config{
		AppName:   "CreatorTask Engine API",
		BodyLimit: int(cfg.MaxUploadSize),
	})

	// Global Middlewares
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(middleware.SetupCORS(cfg.AllowedOrigins))
	app.Use(middleware.SetupRateLimiter(cfg.RateLimitMax))

	// Static files serving with security headers
	app.Use("/uploads", func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		return c.Next()
	})
	app.Static("/uploads", cfg.UploadDir)

	// Root API Group
	api := app.Group("/api/v1")

	// 1. AUTH ROUTES
	authGroup := api.Group("/auth")
	authGroup.Post("/login", middleware.SetupAuthRateLimiter(), authHandler.Login)
	authGroup.Post("/quick-login", authHandler.QuickLogin)
	authGroup.Get("/quick-users", authHandler.GetQuickUsers)

	// 2. CREATOR MANAGEMENT ROUTES (Protected by JWT Auth)
	creator := api.Group("/creator", middleware.RequireJWT(cfg.JWTSecret))

	// Dashboard & Period Options
	creator.Get("/dashboard", reportHandler.GetDashboard)
	creator.Get("/months", reportHandler.GetAvailableMonths)

	// Media Upload
	creator.Post("/upload", uploadHandler.UploadImage)

	// Tasks
	creator.Get("/tasks", taskHandler.GetTasks)
	creator.Post("/tasks", taskHandler.CreateTask)
	creator.Get("/tasks/:id", taskHandler.GetTaskByID)
	creator.Put("/tasks/:id/status", taskHandler.UpdateTaskStatus)
	creator.Post("/tasks/:id/checklist/:itemId/toggle", taskHandler.ToggleChecklistItem)
	creator.Post("/tasks/:id/pay", taskHandler.MarkTaskPaid)
	creator.Delete("/tasks/:id", taskHandler.DeleteTask)

	// Invoices
	creator.Get("/invoices", invoiceHandler.GetInvoices)
	creator.Post("/invoices", invoiceHandler.CreateInvoice)
	creator.Get("/invoices/:id", invoiceHandler.GetInvoiceByID)
	creator.Patch("/invoices/:id/toggle-tax", invoiceHandler.ToggleInvoiceTax)
	creator.Post("/invoices/:id/pay", invoiceHandler.MarkInvoicePaid)

	// Cashflow
	creator.Get("/cashflow", cashflowHandler.GetCashflow)
	creator.Post("/cashflow", cashflowHandler.CreateTransaction)
	creator.Delete("/cashflow/:id", cashflowHandler.DeleteTransaction)

	// Reports
	creator.Get("/reports", reportHandler.GetReports)

	// Settings & Security
	creator.Get("/settings", settingHandler.GetSettings)
	creator.Put("/settings", settingHandler.UpdateSettings)
	creator.Put("/change-password", settingHandler.ChangePassword)

	// User Management (Super Admin)
	creator.Get("/users", settingHandler.GetUsers)
	creator.Post("/users", settingHandler.CreateUser)
	creator.Put("/users/:id", settingHandler.UpdateUser)
	creator.Delete("/users/:id", settingHandler.DeleteUser)

	// Role Management
	creator.Get("/roles", settingHandler.GetRoles)
	creator.Post("/roles", settingHandler.CreateRole)
	creator.Put("/roles/:id", settingHandler.UpdateRole)
	creator.Delete("/roles/:id", settingHandler.DeleteRole)

	// Audit Logs (Immutable Query Only)
	creator.Get("/audit-logs", auditHandler.GetAuditLogs)

	// Start Server
	listenAddr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Starting CreatorTask API Server on %s", listenAddr)
	if err := app.Listen(listenAddr); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
}
