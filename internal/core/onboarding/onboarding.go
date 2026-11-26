// Package onboarding holds the business logic for setting up the system:
// creating branches, adding cars to a branch's fleet, configuring per-category
// pricing, and registering users. Booking itself lives in the booking package.
package onboarding

import (
	"context"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/logger"
	"go-starter/internal/models"

	"github.com/pkg/errors"
)

// Service is the onboarding API used by controllers.
type Service interface {
	CreateBranch(ctx context.Context, req CreateBranchRequest) (models.Branch, error)
	CreateCar(ctx context.Context, req CreateCarRequest) (models.Car, error)
	SetPrice(ctx context.Context, req SetPriceRequest) (models.PriceConfiguration, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (models.User, error)
}

// CreateBranchRequest is the input for onboarding a branch.
type CreateBranchRequest struct {
	Name string
}

// CreateCarRequest adds one car of a category to a branch's fleet.
type CreateCarRequest struct {
	BranchID int64
	Category models.CarCategory
}

// SetPriceRequest configures the per-hour price for a (branch, category) pair.
// Amount is in the smallest currency unit.
type SetPriceRequest struct {
	BranchID int64
	Category models.CarCategory
	Amount   int64
}

// CreateUserRequest registers a user.
type CreateUserRequest struct {
	Name string
}

type service struct {
	repo   internal.Repository
	logger logger.Logger
}

// NewService builds the onboarding service.
func NewService(repo internal.Repository, loggerObj logger.Logger) Service {
	return &service{repo: repo, logger: loggerObj}
}

func (s *service) CreateBranch(ctx context.Context, req CreateBranchRequest) (models.Branch, error) {
	if req.Name == "" {
		return models.Branch{}, errors.Wrap(constants.ErrValidation, "branch name is required")
	}
	return s.repo.CreateBranch(ctx, models.Branch{Name: req.Name})
}

func (s *service) CreateCar(ctx context.Context, req CreateCarRequest) (models.Car, error) {
	if req.BranchID <= 0 {
		return models.Car{}, errors.Wrap(constants.ErrValidation, "branch_id is required")
	}
	if !req.Category.IsValid() {
		return models.Car{}, errors.Wrapf(constants.ErrValidation, "invalid car category %q", req.Category)
	}

	// The branch must exist before we can add a car to its fleet.
	if _, err := s.repo.GetBranchByID(ctx, req.BranchID); err != nil {
		return models.Car{}, err
	}

	return s.repo.CreateCar(ctx, models.Car{
		BranchID: req.BranchID,
		Category: req.Category,
	})
}

func (s *service) SetPrice(ctx context.Context, req SetPriceRequest) (models.PriceConfiguration, error) {
	if req.BranchID <= 0 {
		return models.PriceConfiguration{}, errors.Wrap(constants.ErrValidation, "branch_id is required")
	}
	if !req.Category.IsValid() {
		return models.PriceConfiguration{}, errors.Wrapf(constants.ErrValidation, "invalid car category %q", req.Category)
	}
	if req.Amount <= 0 {
		return models.PriceConfiguration{}, errors.Wrap(constants.ErrValidation, "amount must be positive")
	}

	if _, err := s.repo.GetBranchByID(ctx, req.BranchID); err != nil {
		return models.PriceConfiguration{}, err
	}

	return s.repo.UpsertPriceConfiguration(ctx, models.PriceConfiguration{
		BranchID: req.BranchID,
		Category: req.Category,
		Amount:   req.Amount,
		Unit:     models.UnitPerHour,
	})
}

func (s *service) CreateUser(ctx context.Context, req CreateUserRequest) (models.User, error) {
	if req.Name == "" {
		return models.User{}, errors.Wrap(constants.ErrValidation, "user name is required")
	}
	return s.repo.CreateUser(ctx, models.User{Name: req.Name})
}
