package repository

import (
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (c client) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	row := fromUser(user)
	if err := c.db.WithContext(ctx).Table(Users.ToString()).Create(&row).Error; err != nil {
		return models.User{}, errors.Wrap(err, "create user")
	}
	return row.ToEntity(), nil
}

func (c client) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	var row userRow
	err := c.db.WithContext(ctx).Table(Users.ToString()).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, constants.RecordDoesntExists
	}
	if err != nil {
		return models.User{}, errors.Wrap(err, "get user by id")
	}
	return row.ToEntity(), nil
}
