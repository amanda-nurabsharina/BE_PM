package handler

import (
	"strconv"

	"be-pm/internal/dto"
	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type CashflowHandler struct {
	svc service.Service
}

func NewCashflowHandler(svc service.Service) *CashflowHandler {
	return &CashflowHandler{svc: svc}
}

func (h *CashflowHandler) GetCashflow(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))
	txType := c.Query("type", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "4"))

	data, err := h.svc.GetCashflow(user.UserID, month, year, txType, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal mengambil data buku kas", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Data transaksi kas", data)
}

func (h *CashflowHandler) CreateTransaction(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.TransactionCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	tx, err := h.svc.CreateTransaction(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Transaksi kas berhasil dicatat", tx)
}

func (h *CashflowHandler) DeleteTransaction(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	txID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID transaksi tidak valid", nil)
	}

	if err := h.svc.DeleteTransaction(user.UserID, uint(txID)); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Transaksi berhasil dihapus", nil)
}
