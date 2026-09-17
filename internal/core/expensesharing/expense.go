package expensesharing

import (
	"context"

	"go-starter/internal/constants"
	"go-starter/internal/logger"
	"go-starter/internal/models"
)

// AddExpenseRequest is the validated input for recording an expense. It mirrors
// the Add_Expense command across all three split types:
//
//	Add_Expense [Name] [totalAmount] [paidBy] [createdBy] [totalUsers] Equal   [u1..un]
//	Add_Expense [Name] [totalAmount] [paidBy] [createdBy] [totalUsers] Exact   [u1..un] [amt1..amtn]
//	Add_Expense [Name] [totalAmount] [paidBy] [createdBy] [totalUsers] Percent [u1..un] [pct1..pctn]
//
// All money is paisa. Percent shares are basis points (percent × 100), so 100%
// is 10000 — exact integer arithmetic, no floats.
type AddExpenseRequest struct {
	Name            string
	TotalPaisa      int64
	PaidByUserID    int64
	CreatedByUserID int64
	SplitType       models.SplitType
	Participants    []int64
	// Shares is type-specific: nil/ignored for EQUAL, exact amounts (paisa) for
	// EXACT, percent basis points for PERCENT.
	Shares []int64
}

// AddExpense validates the request, computes each participant's owed share via
// the split strategy, and persists the expense with its splits atomically.
//
// The payer's split carries paid = the full total (single-payer per the command
// grammar); everyone else's paid is 0. Owed is set per the strategy.
func (s *service) AddExpense(ctx context.Context, req AddExpenseRequest) (models.Expense, error) {
	if err := s.validateAddExpense(ctx, req); err != nil {
		return models.Expense{}, err
	}

	strategy, err := strategyFor(req.SplitType)
	if err != nil {
		return models.Expense{}, err
	}

	owed, err := strategy.Split(req.TotalPaisa, req.Participants, req.Shares)
	if err != nil {
		return models.Expense{}, err
	}

	splits := make([]models.Split, len(req.Participants))
	for i, userID := range req.Participants {
		var paid int64
		if userID == req.PaidByUserID {
			paid = req.TotalPaisa
		}
		splits[i] = models.Split{
			UserID:    userID,
			OwedPaisa: owed[i],
			PaidPaisa: paid,
		}
	}

	expense := models.Expense{
		CreatedByUserID: req.CreatedByUserID,
		Name:            req.Name,
		AmountPaisa:     req.TotalPaisa,
		SplitType:       req.SplitType,
	}

	created, err := s.repo.CreateExpenseWithSplits(ctx, expense, splits)
	if err != nil {
		s.logger.Error(ctx, "ExpenseSharing", "AddExpense", "failed to create expense",
			logger.NewString("name", req.Name), logger.NewError(err))
		return models.Expense{}, err
	}
	return created, nil
}

// validateAddExpense runs the structural checks that are independent of the
// split type. Per-type share checks (sum-to-total, sum-to-100) live in the
// strategy.
func (s *service) validateAddExpense(ctx context.Context, req AddExpenseRequest) error {
	switch {
	case req.Name == "":
		return constants.ErrEmptyExpenseName
	case req.TotalPaisa <= 0:
		return constants.ErrNonPositiveAmount
	case len(req.Participants) == 0:
		return constants.ErrEmptyUserList
	}

	// No duplicate participants.
	seen := make(map[int64]struct{}, len(req.Participants))
	for _, id := range req.Participants {
		if _, dup := seen[id]; dup {
			return constants.ErrDuplicateUserInSplit
		}
		seen[id] = struct{}{}
	}

	// The payer must be one of the participants.
	if _, ok := seen[req.PaidByUserID]; !ok {
		return constants.ErrPayerNotInSplit
	}

	// Every referenced user (participants + creator) must exist. The creator may
	// not be a participant, so check it explicitly too.
	if err := s.ensureUsersExist(ctx, req.Participants, req.CreatedByUserID); err != nil {
		return err
	}
	return nil
}

// ensureUsersExist verifies each user id resolves. A missing user surfaces as
// constants.RecordDoesntExists from the repository, which is returned as-is.
func (s *service) ensureUsersExist(ctx context.Context, participants []int64, createdBy int64) error {
	ids := make(map[int64]struct{}, len(participants)+1)
	for _, id := range participants {
		ids[id] = struct{}{}
	}
	ids[createdBy] = struct{}{}

	for id := range ids {
		if _, err := s.repo.GetUserByID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
