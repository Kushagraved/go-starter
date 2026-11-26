package controller

import (
	"net/http"
	"strconv"

	apionboarding "go-starter/api/onboarding"
	"go-starter/internal/constants"
	"go-starter/internal/core/onboarding"
	"go-starter/internal/dto"
	"go-starter/internal/models"

	"github.com/gin-gonic/gin"
)

func (s *Server) CreateBranch(ctx *gin.Context) {
	var req apionboarding.CreateBranchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}

	branch, err := s.services.Onboarding.CreateBranch(ctx.Request.Context(), onboarding.CreateBranchRequest{
		Name: req.Name,
	})
	if err != nil {
		s.RespondError(ctx, "CreateBranch", err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"success": true, "data": dto.ToBranchResponse(branch)})
}

func (s *Server) CreateCar(ctx *gin.Context) {
	branchID, err := parseInt64Param(ctx, "branch_id")
	if err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "invalid branch_id")
		return
	}

	var req apionboarding.CreateCarRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}

	car, err := s.services.Onboarding.CreateCar(ctx.Request.Context(), onboarding.CreateCarRequest{
		BranchID: branchID,
		Category: models.CarCategory(req.Category),
	})
	if err != nil {
		s.RespondError(ctx, "CreateCar", err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"success": true, "data": dto.ToCarResponse(car)})
}

func (s *Server) SetPrice(ctx *gin.Context) {
	branchID, err := parseInt64Param(ctx, "branch_id")
	if err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, "invalid branch_id")
		return
	}

	var req apionboarding.SetPriceRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}

	cfg, err := s.services.Onboarding.SetPrice(ctx.Request.Context(), onboarding.SetPriceRequest{
		BranchID: branchID,
		Category: models.CarCategory(req.Category),
		Amount:   req.Amount,
	})
	if err != nil {
		s.RespondError(ctx, "SetPrice", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"success": true, "data": dto.ToPriceConfigurationResponse(cfg)})
}

func (s *Server) CreateUser(ctx *gin.Context) {
	var req apionboarding.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ThrowError(ctx, http.StatusBadRequest, constants.ErrBadRequest, err.Error())
		return
	}

	user, err := s.services.Onboarding.CreateUser(ctx.Request.Context(), onboarding.CreateUserRequest{
		Name: req.Name,
	})
	if err != nil {
		s.RespondError(ctx, "CreateUser", err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"success": true, "data": dto.ToUserResponse(user)})
}

// parseInt64Param reads and parses a path parameter as int64.
func parseInt64Param(ctx *gin.Context, name string) (int64, error) {
	return strconv.ParseInt(ctx.Param(name), 10, 64)
}
