package handler

import (
	"strconv"

	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type ReportHandler struct {
	svc service.Service
}

func NewReportHandler(svc service.Service) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) GetReports(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))
	coopType := c.Query("type", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "4"))

	data, err := h.svc.GetReports(user.UserID, month, year, coopType, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal menyusun laporan rekapitulasi", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Laporan rekapitulasi proyek", data)
}

func (h *ReportHandler) GetDashboard(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	data, err := h.svc.GetDashboard(user.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat ringkasan dashboard", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Ringkasan metrik dashboard", data)
}

func (h *ReportHandler) GetAvailableMonths(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	data, err := h.svc.GetAvailableMonths(user.UserID)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal memuat daftar bulan", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar opsi bulan dinamis", data)
}
