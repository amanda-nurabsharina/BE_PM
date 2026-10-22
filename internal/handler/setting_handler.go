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
