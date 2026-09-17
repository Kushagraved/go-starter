package repository

import (
	"time"

	"go-starter/internal/models"
)

// userRow is the database shape of a user. It never leaves this package; callers
// receive models.User via ToEntity.
type userRow struct {
	ID        int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	Phone     string    `gorm:"column:phone"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (r userRow) ToEntity() models.User {
	return models.User{
		ID:        r.ID,
		Name:      r.Name,
		Email:     r.Email,
		Phone:     r.Phone,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// fromUser omits ID so Postgres assigns the bigserial value.
func fromUser(u models.User) userRow {
	return userRow{Name: u.Name, Email: u.Email, Phone: u.Phone}
}

// expenseRow is the database shape of an expense.
type expenseRow struct {
	ID              int64     `gorm:"column:id"` // bigserial; DB assigns on insert
	CreatedByUserID int64     `gorm:"column:created_by_user_id"`
	Name            string    `gorm:"column:name"`
	AmountPaisa     int64     `gorm:"column:amount_paisa"`
	SplitType       string    `gorm:"column:split_type"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (r expenseRow) ToEntity() models.Expense {
	return models.Expense{
		ID:              r.ID,
		CreatedByUserID: r.CreatedByUserID,
		Name:            r.Name,
		AmountPaisa:     r.AmountPaisa,
		SplitType:       models.SplitType(r.SplitType),
		CreatedAt:       r.CreatedAt,
	}
}

// fromExpense omits ID so Postgres assigns the bigserial value.
func fromExpense(e models.Expense) expenseRow {
	return expenseRow{
		CreatedByUserID: e.CreatedByUserID,
		Name:            e.Name,
		AmountPaisa:     e.AmountPaisa,
		SplitType:       string(e.SplitType),
	}
}

// expenseSplitRow is the database shape of one user's stake in one expense.
type expenseSplitRow struct {
	ID        int64 `gorm:"column:id"` // bigserial; DB assigns on insert
	ExpenseID int64 `gorm:"column:expense_id"`
	UserID    int64 `gorm:"column:user_id"`
	OwedPaisa int64 `gorm:"column:owed_paisa"`
	PaidPaisa int64 `gorm:"column:paid_paisa"`
}

func (r expenseSplitRow) ToEntity() models.Split {
	return models.Split{
		ID:        r.ID,
		ExpenseID: r.ExpenseID,
		UserID:    r.UserID,
		OwedPaisa: r.OwedPaisa,
		PaidPaisa: r.PaidPaisa,
	}
}

// fromSplit omits ID so Postgres assigns the bigserial value.
func fromSplit(s models.Split) expenseSplitRow {
	return expenseSplitRow{
		ExpenseID: s.ExpenseID,
		UserID:    s.UserID,
		OwedPaisa: s.OwedPaisa,
		PaidPaisa: s.PaidPaisa,
	}
}
