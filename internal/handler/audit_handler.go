package handler

import (
	"strconv"

	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type AuditHandler struct {
	svc service.Service
}

func NewAuditHandler(svc service.Service) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) GetAuditLogs(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	module := c.Query("module", "all")
	search := c.Query("search", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "15"))

	data, err := h.svc.GetAuditLogs(user.UserID, module, search, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat log audit", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar jejak audit aktivitas", data)
}
