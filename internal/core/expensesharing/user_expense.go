package expensesharing

import (
	"context"

	"go-starter/internal/logger"
)

// GetUserExpenses returns the expenses the user participates in, each annotated
// with that user's owed and paid amounts.
//
// The user must exist first: a Show_User_Expense on an unknown id is an error,
// not an empty list, so the caller can tell "no expenses" apart from "no such
// user".
func (s *service) GetUserExpenses(ctx context.Context, userID int64) ([]UserExpense, error) {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	expenses, err := s.repo.ListExpensesByUserID(ctx, userID)
	if err != nil {
		s.logger.Error(ctx, "ExpenseSharing", "GetUserExpenses", "failed to list expenses",
			logger.NewInt64("user_id", userID), logger.NewError(err))
		return nil, err
	}

	out := make([]UserExpense, 0, len(expenses))
	for _, exp := range expenses {
		var owed, paid int64
		for _, sp := range exp.Splits {
			if sp.UserID == userID {
				owed = sp.OwedPaisa
				paid = sp.PaidPaisa
				break
			}
		}
		out = append(out, UserExpense{
			Expense:       exp,
			UserOwedPaisa: owed,
			UserPaidPaisa: paid,
		})
	}
	return out, nil
}
