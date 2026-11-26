package repository

import (
	"context"
	"encoding/json"
	"time"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/*
UpsertPriceConfiguration sets the price for a (branch, category) pair.

When a configuration already exists and the amount changes, a models.PriceChange
is appended to the JSONB audit trail. The read-modify-write is wrapped in its own
transaction (when not already inside one via Atomic) and locks the existing row
FOR UPDATE so two concurrent price changes can't clobber each other's trail.
*/
func (c client) UpsertPriceConfiguration(ctx context.Context, cfg models.PriceConfiguration) (models.PriceConfiguration, error) {
	unit := cfg.Unit
	if unit == "" {
		unit = models.UnitPerHour
	}

	err := c.Atomic(ctx, func(store internal.DAO) error {
		txClient := store.(client)
		// tx returns a fresh, un-polluted session per statement. Reusing one
		// chained *gorm.DB across First/Create/Updates leaks clauses (e.g. the
		// FOR UPDATE lock and WHERE conditions) from one call into the next.
		tx := func() *gorm.DB {
			return txClient.db.WithContext(ctx).Table(PriceConfigurations.ToString())
		}

		var existing priceConfigurationRow
		findErr := tx().Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("branch_id = ? AND category = ?", cfg.BranchID, cfg.Category.String()).
			First(&existing).Error

		switch {
		case errors.Is(findErr, gorm.ErrRecordNotFound):
			// First time this (branch, category) is priced: insert with an empty trail.
			row := priceConfigurationRow{
				BranchID:   cfg.BranchID,
				Category:   cfg.Category.String(),
				Amount:     cfg.Amount,
				Unit:       string(unit),
				AuditTrail: "[]",
			}
			if err := tx().Create(&row).Error; err != nil {
				return errors.Wrap(err, "insert price configuration")
			}
			return nil

		case findErr != nil:
			return errors.Wrap(findErr, "lock price configuration")
		}

		// Existing row: append to the audit trail only when the amount actually moves.
		trail := existing.ToEntity().AuditTrail
		if existing.Amount != cfg.Amount {
			trail = append(trail, models.PriceChange{
				UpdatedAt:  time.Now().UTC(),
				FromAmount: existing.Amount,
				ToAmount:   cfg.Amount,
			})
		}
		trailJSON, err := json.Marshal(trail)
		if err != nil {
			return errors.Wrap(err, "marshal audit trail")
		}

		updateErr := tx().Where("id = ?", existing.ID).
			Updates(map[string]interface{}{
				"amount":      cfg.Amount,
				"unit":        string(unit),
				"audit_trail": string(trailJSON),
				"updated_at":  time.Now().UTC(),
			}).Error
		if updateErr != nil {
			return errors.Wrap(updateErr, "update price configuration")
		}
		return nil
	})
	if err != nil {
		return models.PriceConfiguration{}, err
	}

	return c.GetPriceConfiguration(ctx, cfg.BranchID, cfg.Category)
}

func (c client) GetPriceConfiguration(ctx context.Context, branchID int64, category models.CarCategory) (models.PriceConfiguration, error) {
	var row priceConfigurationRow
	err := c.db.WithContext(ctx).Table(PriceConfigurations.ToString()).
		Where("branch_id = ? AND category = ?", branchID, category.String()).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.PriceConfiguration{}, constants.PriceConfigNotFound
	}
	if err != nil {
		return models.PriceConfiguration{}, errors.Wrap(err, "get price configuration")
	}
	return row.ToEntity(), nil
}
