package repository

import (
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (c client) CreateBranch(ctx context.Context, branch models.Branch) (models.Branch, error) {
	row := fromBranch(branch)
	if err := c.db.WithContext(ctx).Table(Branches.ToString()).Create(&row).Error; err != nil {
		return models.Branch{}, errors.Wrap(err, "create branch")
	}
	return row.ToEntity(), nil
}

func (c client) GetBranchByID(ctx context.Context, id int64) (models.Branch, error) {
	var row branchRow
	err := c.db.WithContext(ctx).Table(Branches.ToString()).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Branch{}, constants.RecordDoesntExists
	}
	if err != nil {
		return models.Branch{}, errors.Wrap(err, "get branch by id")
	}
	return row.ToEntity(), nil
}
