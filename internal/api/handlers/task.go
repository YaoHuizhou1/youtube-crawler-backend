package handlers

import (
	"github.com/example/youtube-dialogue-crawler/internal/models"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/response"
	"github.com/example/youtube-dialogue-crawler/internal/pkg/websocket"
	"github.com/example/youtube-dialogue-crawler/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

type TaskHandler struct {
	repo   *repository.TaskRepository
	client *asynq.Client
	hub    *websocket.Hub
}

func NewTaskHandler(repo *repository.TaskRepository, client *asynq.Client, hub *websocket.Hub) *TaskHandler {
	return &TaskHandler{repo: repo, client: client, hub: hub}
}

func (h *TaskHandler) Create(c *gin.Context) {
	var req models.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	task, err := h.repo.Create(c.Request.Context(), &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	h.hub.Broadcast("task_created", task)
	response.Created(c, task)
}

func (h *TaskHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if task == nil {
		response.NotFound(c, "Task not found")
		return
	}

	response.Success(c, task)
}

func (h *TaskHandler) List(c *gin.Context) {
	var params models.TaskListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	tasks, total, err := h.repo.List(c.Request.Context(), &params)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.SuccessWithPage(c, tasks, params.Page, params.PageSize, total)
}

func (h *TaskHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	var req models.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	task, err := h.repo.Update(c.Request.Context(), id, &req)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if task == nil {
		response.NotFound(c, "Task not found")
		return
	}

	h.hub.Broadcast("task_updated", task)
	response.Success(c, task)
}

func (h *TaskHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	h.hub.Broadcast("task_deleted", map[string]string{"id": idStr})
	response.NoContent(c)
}

func (h *TaskHandler) Start(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	task, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if task == nil {
		response.NotFound(c, "Task not found")
		return
	}

	if task.Status == models.TaskStatusRunning {
		response.BadRequest(c, "Task is already running")
		return
	}

	// Update status to running
	if err := h.repo.UpdateStatus(c.Request.Context(), id, models.TaskStatusRunning, nil); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	// Enqueue discovery task
	asynqTask := asynq.NewTask("discovery:run", []byte(idStr))
	_, err = h.client.Enqueue(asynqTask)
	if err != nil {
		h.repo.UpdateStatus(c.Request.Context(), id, models.TaskStatusFailed, &[]string{err.Error()}[0])
		response.InternalError(c, "Failed to enqueue task")
		return
	}

	task.Status = models.TaskStatusRunning
	h.hub.Broadcast("task_started", task)
	response.Success(c, task)
}

func (h *TaskHandler) Pause(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), id, models.TaskStatusPaused, nil); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	task, _ := h.repo.GetByID(c.Request.Context(), id)
	h.hub.Broadcast("task_paused", task)
	response.Success(c, task)
}

func (h *TaskHandler) Stop(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "Invalid task ID")
		return
	}

	if err := h.repo.UpdateStatus(c.Request.Context(), id, models.TaskStatusCompleted, nil); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	task, _ := h.repo.GetByID(c.Request.Context(), id)
	h.hub.Broadcast("task_stopped", task)
	response.Success(c, task)
}
