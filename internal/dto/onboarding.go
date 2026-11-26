// Package dto maps domain entities (models.*) to the api.* response shapes. It is
// the one place that knows both sides, keeping controllers thin and models clean.
package dto

import (
	"time"

	apionboarding "go-starter/api/onboarding"
	"go-starter/internal/models"
)

func ToBranchResponse(b models.Branch) apionboarding.BranchResponse {
	return apionboarding.BranchResponse{ID: b.ID, Name: b.Name}
}

func ToCarResponse(c models.Car) apionboarding.CarResponse {
	return apionboarding.CarResponse{
		ID:       c.ID,
		BranchID: c.BranchID,
		Category: c.Category.String(),
	}
}

func ToUserResponse(u models.User) apionboarding.UserResponse {
	return apionboarding.UserResponse{ID: u.ID, Name: u.Name}
}

func ToPriceConfigurationResponse(p models.PriceConfiguration) apionboarding.PriceConfigurationResponse {
	trail := make([]apionboarding.PriceChangeResponse, 0, len(p.AuditTrail))
	for _, ch := range p.AuditTrail {
		trail = append(trail, apionboarding.PriceChangeResponse{
			UpdatedAt:  ch.UpdatedAt.Format(time.RFC3339),
			FromAmount: ch.FromAmount,
			ToAmount:   ch.ToAmount,
		})
	}
	return apionboarding.PriceConfigurationResponse{
		ID:         p.ID,
		BranchID:   p.BranchID,
		Category:   p.Category.String(),
		Amount:     p.Amount,
		Unit:       string(p.Unit),
		AuditTrail: trail,
	}
}
