package repository

import (
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (c client) CreateCar(ctx context.Context, car models.Car) (models.Car, error) {
	row := fromCar(car)
	if err := c.db.WithContext(ctx).Table(Cars.ToString()).Create(&row).Error; err != nil {
		return models.Car{}, errors.Wrap(err, "create car")
	}
	return row.ToEntity(), nil
}

func (c client) GetCarByID(ctx context.Context, id int64) (models.Car, error) {
	var row carRow
	err := c.db.WithContext(ctx).Table(Cars.ToString()).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Car{}, constants.RecordDoesntExists
	}
	if err != nil {
		return models.Car{}, errors.Wrap(err, "get car by id")
	}
	return row.ToEntity(), nil
}
