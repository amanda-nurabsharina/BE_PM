package handler

import (
	"be-pm/internal/dto"
	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type SettingHandler struct {
	svc service.Service
}

func NewSettingHandler(svc service.Service) *SettingHandler {
	return &SettingHandler{svc: svc}
}

func (h *SettingHandler) GetSettings(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	settings, err := h.svc.GetSettings(user.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Pengaturan pengguna tidak ditemukan", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Pengaturan profil studio", settings)
}

func (h *SettingHandler) UpdateSettings(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.SettingsUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	updated, err := h.svc.UpdateSettings(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pengaturan studio berhasil diperbarui", updated)
}

func (h *SettingHandler) ChangePassword(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.svc.ChangePassword(user.UserID, req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Password berhasil diperbarui", nil)
}

// User Management Handlers (Super Admin Only)
func (h *SettingHandler) GetUsers(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	users, err := h.svc.GetUsers(user.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusForbidden, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Daftar pengguna studio", users)
}

func (h *SettingHandler) CreateUser(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.UserCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.svc.CreateUser(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Pengguna baru berhasil dibuat", res)
}

func (h *SettingHandler) UpdateUser(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	targetID, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID pengguna tidak valid", nil)
	}

	var req dto.UserUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.svc.UpdateUser(user.UserID, uint(targetID), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Data pengguna berhasil diperbarui", res)
}

func (h *SettingHandler) DeleteUser(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	targetID, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID pengguna tidak valid", nil)
	}

	if err := h.svc.DeleteUser(user.UserID, uint(targetID)); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pengguna berhasil dihapus", nil)
}

// Role Management Handlers
func (h *SettingHandler) GetRoles(c *fiber.Ctx) error {
	roles, err := h.svc.GetRoles()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat roles", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar role sistem", roles)
}

func (h *SettingHandler) CreateRole(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.svc.CreateRole(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Role baru berhasil dibuat", res)
}

func (h *SettingHandler) UpdateRole(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID role tidak valid", nil)
	}

	var req dto.RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	res, err := h.svc.UpdateRole(user.UserID, uint(id), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Role berhasil diperbarui", res)
}

func (h *SettingHandler) DeleteRole(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	id, err := c.ParamsInt("id")
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID role tidak valid", nil)
	}

	if err := h.svc.DeleteRole(user.UserID, uint(id)); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Role berhasil dihapus", nil)
}
