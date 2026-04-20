package controller

import (
	"net/http"

	apitask "go-starter/api/task"
	"go-starter/internal/constants"
	"go-starter/internal/core/task"
	"go-starter/internal/dto"
	"go-starter/internal/logger"
	"go-starter/internal/models"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateTask(ctx *gin.Context) {
	var req apitask.CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		throwError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}
	created, err := s.services.Task.Create(ctx.Request.Context(), task.CreateRequest{
		Title:       req.Title,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		s.logger.Error(ctx.Request.Context(), "task", "CreateTask", "failed to create task", logger.Err(err))
		throwError(ctx, http.StatusInternalServerError, constants.ErrInternalServer, err.Error())
		return
	}
	ctx.JSON(http.StatusCreated, dto.ToTaskResponse(created))
}

func (s *Server) GetTask(ctx *gin.Context) {
	id := ctx.Param("id")
	t, err := s.services.Task.Get(ctx.Request.Context(), id)
	if err != nil {
		throwError(ctx, http.StatusNotFound, constants.ErrNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, dto.ToTaskResponse(t))
}

func (s *Server) ListTasks(ctx *gin.Context) {
	var filters []models.RepoFilter
	if status := ctx.Query("status"); status != "" {
		filters = append(filters, models.RepoFilter{Key: "status", Value: status})
	}
	if assignee := ctx.Query("assignee_id"); assignee != "" {
		filters = append(filters, models.RepoFilter{Key: "assignee_id", Value: assignee})
	}
	tasks, err := s.services.Task.List(ctx.Request.Context(), filters)
	if err != nil {
		s.logger.Error(ctx.Request.Context(), "task", "ListTasks", "failed to list tasks", logger.Err(err))
		throwError(ctx, http.StatusInternalServerError, constants.ErrInternalServer, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, dto.ToTaskListResponse(tasks))
}

func (s *Server) UpdateTask(ctx *gin.Context) {
	id := ctx.Param("id")
	var req apitask.UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		throwError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}
	updated, err := s.services.Task.Update(ctx.Request.Context(), id, task.UpdateRequest{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		throwError(ctx, http.StatusInternalServerError, constants.ErrInternalServer, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, dto.ToTaskResponse(updated))
}

func (s *Server) DeleteTask(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := s.services.Task.Delete(ctx.Request.Context(), id); err != nil {
		throwError(ctx, http.StatusNotFound, constants.ErrNotFound, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true})
}
