package handler

import (
	"strconv"

	"be-pm/internal/dto"
	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type TenantHandler struct {
	svc service.Service
}

func NewTenantHandler(svc service.Service) *TenantHandler {
	return &TenantHandler{svc: svc}
}

func (h *TenantHandler) checkAdminAccess(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	if user == nil || user.Role != "SUPER_ADMIN" {
		return response.Error(c, fiber.StatusForbidden, "Akses ditolak: hanya Administrator yang memiliki wewenang mengelola tenant", nil)
	}
	return nil
}

func (h *TenantHandler) GetTenants(c *fiber.Ctx) error {
	if err := h.checkAdminAccess(c); err != nil {
		return err
	}

	tenants, err := h.svc.GetTenants()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat daftar tenant", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar tenant berhasil dimuat", tenants)
}

func (h *TenantHandler) GetTenantByID(c *fiber.Ctx) error {
	if err := h.checkAdminAccess(c); err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tenant tidak valid", nil)
	}

	tenant, err := h.svc.GetTenantByID(uint(id))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Detail tenant berhasil dimuat", tenant)
}

func (h *TenantHandler) CreateTenant(c *fiber.Ctx) error {
	if err := h.checkAdminAccess(c); err != nil {
		return err
	}

	var req dto.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Format payload tidak valid", err.Error())
	}

	tenant, err := h.svc.CreateTenant(req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Tenant dan owner berhasil dibuat", tenant)
}

func (h *TenantHandler) UpdateTenantStatus(c *fiber.Ctx) error {
	if err := h.checkAdminAccess(c); err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tenant tidak valid", nil)
	}

	var req dto.UpdateTenantStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Format payload tidak valid", err.Error())
	}

	tenant, err := h.svc.UpdateTenantStatus(uint(id), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Status tenant berhasil diperbarui", tenant)
}

func (h *TenantHandler) ResetTenantOwnerPassword(c *fiber.Ctx) error {
	if err := h.checkAdminAccess(c); err != nil {
		return err
	}

	idParam := c.Params("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tenant tidak valid", nil)
	}

	var req dto.ResetTenantOwnerPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Format payload tidak valid", err.Error())
	}

	if err := h.svc.ResetTenantOwnerPassword(uint(id), req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Password owner tenant berhasil direset", nil)
}

func (h *TenantHandler) AdminLogin(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	res, err := h.svc.AdminLogin(req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Admin login berhasil", res)
}

