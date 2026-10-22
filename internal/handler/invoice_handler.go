package handler

import (
	"strconv"

	"be-pm/internal/dto"
	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type InvoiceHandler struct {
	svc service.Service
}

func NewInvoiceHandler(svc service.Service) *InvoiceHandler {
	return &InvoiceHandler{svc: svc}
}

func (h *InvoiceHandler) GetInvoices(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))
	status := c.Query("status", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "4"))

	data, err := h.svc.GetInvoices(user.UserID, month, year, status, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal mengambil daftar tagihan", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar tagihan invoice", data)
}

func (h *InvoiceHandler) GetInvoiceByID(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	invID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID invoice tidak valid", nil)
	}

	inv, err := h.svc.GetInvoiceByID(user.UserID, uint(invID))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Invoice tidak ditemukan", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Detail invoice", inv)
}

func (h *InvoiceHandler) CreateInvoice(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.InvoiceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	invoice, err := h.svc.CreateInvoice(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Invoice berhasil diterbitkan", invoice)
}

func (h *InvoiceHandler) ToggleInvoiceTax(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	invID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID invoice tidak valid", nil)
	}

	inv, err := h.svc.ToggleInvoiceTax(user.UserID, uint(invID))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Status PPN invoice berhasil diubah", inv)
}

func (h *InvoiceHandler) MarkInvoicePaid(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	invID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID invoice tidak valid", nil)
	}

	var req dto.PaymentConfirmationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.svc.MarkInvoicePaid(user.UserID, uint(invID), req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Invoice berhasil ditandai lunas dan kas dibukukan otomatis", nil)
}
