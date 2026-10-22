package handler

import (
	"be-pm/internal/dto"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc service.Service
}

func NewAuthHandler(svc service.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request body", err.Error())
	}

	res, err := h.svc.Login(req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Login berhasil", res)
}

func (h *AuthHandler) QuickLogin(c *fiber.Ctx) error {
	var req dto.QuickLoginRequest
	_ = c.BodyParser(&req)

	res, err := h.svc.QuickLogin(req.Email)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Quick login berhasil", res)
}

func (h *AuthHandler) GetQuickUsers(c *fiber.Ctx) error {
	users, err := h.svc.GetQuickUsers()
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat profil cepat", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar profil cepat", users)
}
