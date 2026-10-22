package handler

import (
	"strconv"

	"be-pm/internal/dto"
	"be-pm/internal/middleware"
	"be-pm/internal/service"
	"be-pm/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type TaskHandler struct {
	svc service.Service
}

func NewTaskHandler(svc service.Service) *TaskHandler {
	return &TaskHandler{svc: svc}
}

func (h *TaskHandler) GetTasks(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))
	taskType := c.Query("type", "all")
	status := c.Query("status", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "4"))

	data, err := h.svc.GetTasks(user.UserID, month, year, taskType, status, page, limit)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "Gagal mengambil daftar tugas", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Daftar tugas", data)
}

func (h *TaskHandler) GetTaskByID(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tugas tidak valid", nil)
	}

	task, err := h.svc.GetTaskByID(user.UserID, uint(taskID))
	if err != nil {
		return response.Error(c, fiber.StatusNotFound, "Tugas tidak ditemukan", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Detail tugas", task)
}

func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	var req dto.TaskCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	task, err := h.svc.CreateTask(user.UserID, req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusCreated, "Tugas berhasil dibuat", task)
}

func (h *TaskHandler) UpdateTaskStatus(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tugas tidak valid", nil)
	}

	var req dto.TaskStatusUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	task, err := h.svc.UpdateTaskStatus(user.UserID, uint(taskID), req.Status)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Status tugas berhasil diperbarui", task)
}

func (h *TaskHandler) ToggleChecklistItem(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	taskID, _ := strconv.ParseUint(c.Params("id"), 10, 64)
	itemID, _ := strconv.ParseUint(c.Params("itemId"), 10, 64)

	item, err := h.svc.ToggleChecklistItem(user.UserID, uint(taskID), uint(itemID))
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Gagal memperbarui checklist", err.Error())
	}

	return response.Success(c, fiber.StatusOK, "Status checklist berhasil diperbarui", item)
}

func (h *TaskHandler) MarkTaskPaid(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tugas tidak valid", nil)
	}

	var req dto.PaymentConfirmationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "Invalid request payload", err.Error())
	}

	if err := h.svc.MarkTaskPaid(user.UserID, uint(taskID), req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Pelunasan berhasil diproses dan disinkronkan ke seluruh sistem", nil)
}

func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	user := middleware.GetAuthUser(c)
	taskID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, "ID tugas tidak valid", nil)
	}

	if err := h.svc.DeleteTask(user.UserID, uint(taskID)); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	return response.Success(c, fiber.StatusOK, "Tugas berhasil dihapus", nil)
}
