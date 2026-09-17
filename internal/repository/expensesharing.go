package repository

import (
	"context"
	"strings"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/models"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// CreateUser inserts a user. A unique-constraint violation is translated to a
// domain sentinel by constraint name so the caller never inspects driver errors.
func (c client) CreateUser(ctx context.Context, user models.User) (models.User, error) {
	row := fromUser(user)
	if err := c.db.WithContext(ctx).Table(Users.ToString()).Create(&row).Error; err != nil {
		return models.User{}, translateUserWriteErr(err)
	}
	return row.ToEntity(), nil // row.ID now holds the DB-assigned bigserial
}

// GetUserByID returns the user or constants.RecordDoesntExists.
func (c client) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	var row userRow
	err := c.db.WithContext(ctx).Table(Users.ToString()).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.User{}, constants.RecordDoesntExists
	}
	if err != nil {
		return models.User{}, err
	}
	return row.ToEntity(), nil
}

// ListUsers returns every user ordered by id.
func (c client) ListUsers(ctx context.Context) ([]models.User, error) {
	var rows []userRow
	if err := c.db.WithContext(ctx).Table(Users.ToString()).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	users := make([]models.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, row.ToEntity())
	}
	return users, nil
}

// CreateExpenseWithSplits writes the expense and its splits in one transaction.
// The expense id assigned by the DB is stamped onto each split before insert.
func (c client) CreateExpenseWithSplits(ctx context.Context, expense models.Expense, splits []models.Split) (models.Expense, error) {
	var created models.Expense
	err := c.Atomic(ctx, func(store internal.DAO) error {
		txn := store.(client)

		expRow := fromExpense(expense)
		if err := txn.db.WithContext(ctx).Table(Expenses.ToString()).Create(&expRow).Error; err != nil {
			return err
		}

		splitRows := make([]expenseSplitRow, 0, len(splits))
		for _, s := range splits {
			s.ExpenseID = expRow.ID
			splitRows = append(splitRows, fromSplit(s))
		}
		if len(splitRows) > 0 {
			if err := txn.db.WithContext(ctx).Table(ExpenseSplits.ToString()).Create(&splitRows).Error; err != nil {
				return err
			}
		}

		created = expRow.ToEntity()
		created.Splits = make([]models.Split, 0, len(splitRows))
		for _, sr := range splitRows {
			created.Splits = append(created.Splits, sr.ToEntity())
		}
		return nil
	})
	if err != nil {
		return models.Expense{}, err
	}
	return created, nil
}

// ListAllSplits returns every split across all expenses — the raw input for
// deriving balances.
func (c client) ListAllSplits(ctx context.Context) ([]models.Split, error) {
	var rows []expenseSplitRow
	if err := c.db.WithContext(ctx).Table(ExpenseSplits.ToString()).Find(&rows).Error; err != nil {
		return nil, err
	}
	splits := make([]models.Split, 0, len(rows))
	for _, row := range rows {
		splits = append(splits, row.ToEntity())
	}
	return splits, nil
}

// ListExpensesByUserID returns the expenses the user participates in, each
// hydrated with all of its splits (so the caller can show owed/paid per user).
func (c client) ListExpensesByUserID(ctx context.Context, userID int64) ([]models.Expense, error) {
	var expenseIDs []int64
	err := c.db.WithContext(ctx).
		Table(ExpenseSplits.ToString()).
		Where("user_id = ?", userID).
		Distinct().
		Pluck("expense_id", &expenseIDs).Error
	if err != nil {
		return nil, err
	}
	if len(expenseIDs) == 0 {
		return []models.Expense{}, nil
	}

	var expRows []expenseRow
	if err := c.db.WithContext(ctx).Table(Expenses.ToString()).
		Where("id IN ?", expenseIDs).Order("id").Find(&expRows).Error; err != nil {
		return nil, err
	}

	var splitRows []expenseSplitRow
	if err := c.db.WithContext(ctx).Table(ExpenseSplits.ToString()).
		Where("expense_id IN ?", expenseIDs).Find(&splitRows).Error; err != nil {
		return nil, err
	}

	splitsByExpense := make(map[int64][]models.Split, len(expRows))
	for _, sr := range splitRows {
		splitsByExpense[sr.ExpenseID] = append(splitsByExpense[sr.ExpenseID], sr.ToEntity())
	}

	expenses := make([]models.Expense, 0, len(expRows))
	for _, er := range expRows {
		exp := er.ToEntity()
		exp.Splits = splitsByExpense[er.ID]
		expenses = append(expenses, exp)
	}
	return expenses, nil
}

// translateUserWriteErr maps a unique-constraint violation to a domain sentinel
// by matching the named constraint from scripts/db/schema.sql.
func translateUserWriteErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "uq_users_email"):
		return constants.DuplicateEmail
	case strings.Contains(msg, "uq_users_phone"):
		return constants.DuplicatePhone
	default:
		return err
	}
}
