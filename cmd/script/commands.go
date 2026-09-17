package main

import (
	"context"
	"strconv"
	"strings"

	"go-starter/internal/core/expensesharing"
	"go-starter/internal/models"

	"github.com/pkg/errors"
)

// executor runs parsed commands against the core service and returns the lines
// to print. It holds no state beyond the service, so it is trivial to test with
// a mocked service.
type executor struct {
	svc expensesharing.Service
}

// Run parses and executes one command line, returning the output lines. A blank
// line or a line beginning with '#' is a no-op (allows comments in the file).
func (e *executor) Run(ctx context.Context, line string) ([]string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, nil
	}

	fields := strings.Fields(line)
	command := fields[0]
	args := fields[1:]

	switch command {
	case "Add_User":
		return e.addUser(ctx, args)
	case "Add_Expense":
		return e.addExpense(ctx, args)
	case "Show_User_Data":
		return e.showUserData(ctx, args)
	case "Show_All_User_Balance":
		return e.showAllBalances(ctx, args)
	case "Show_User_Expense":
		return e.showUserExpense(ctx, args)
	default:
		return nil, errors.Errorf("unknown command: %q", command)
	}
}

// Add_User [Name] [Email] [PhoneNumber]
func (e *executor) addUser(ctx context.Context, args []string) ([]string, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Add_User expects 3 arguments (name email phone), got %d", len(args))
	}

	user, err := e.svc.AddUser(ctx, expensesharing.AddUserRequest{
		Name:  args[0],
		Email: args[1],
		Phone: args[2],
	})
	if err != nil {
		return nil, err
	}
	return []string{strconv.FormatInt(user.ID, 10)}, nil
}

// Add_Expense [Name] [totalAmount] [paidBy] [createdBy] [totalUsers] <Type> [users...] [shares...]
func (e *executor) addExpense(ctx context.Context, args []string) ([]string, error) {
	// Fixed prefix: name, total, paidBy, createdBy, totalUsers, type = 6 tokens.
	if len(args) < 6 {
		return nil, errors.Errorf("Add_Expense expects at least 6 arguments, got %d", len(args))
	}

	name := args[0]

	totalPaisa, err := rupeesToPaisa(args[1])
	if err != nil {
		return nil, err
	}
	paidBy, err := parseID(args[2], "paidBy")
	if err != nil {
		return nil, err
	}
	createdBy, err := parseID(args[3], "createdBy")
	if err != nil {
		return nil, err
	}
	totalUsers, err := strconv.Atoi(args[4])
	if err != nil || totalUsers <= 0 {
		return nil, errors.Errorf("invalid totalUsers: %q", args[4])
	}

	splitType, err := parseSplitType(args[5])
	if err != nil {
		return nil, err
	}

	rest := args[6:]
	if len(rest) < totalUsers {
		return nil, errors.Errorf("expected %d user ids, got %d tokens after the split type", totalUsers, len(rest))
	}

	participants := make([]int64, totalUsers)
	for i := 0; i < totalUsers; i++ {
		id, err := parseID(rest[i], "user")
		if err != nil {
			return nil, err
		}
		participants[i] = id
	}

	shareTokens := rest[totalUsers:]
	shares, err := parseShares(splitType, shareTokens, totalUsers)
	if err != nil {
		return nil, err
	}

	expense, err := e.svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
		Name:            name,
		TotalPaisa:      totalPaisa,
		PaidByUserID:    paidBy,
		CreatedByUserID: createdBy,
		SplitType:       splitType,
		Participants:    participants,
		Shares:          shares,
	})
	if err != nil {
		return nil, err
	}
	return []string{"expense " + strconv.FormatInt(expense.ID, 10) + " recorded"}, nil
}

// Show_User_Data [UserId]
func (e *executor) showUserData(ctx context.Context, args []string) ([]string, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Show_User_Data expects 1 argument (userId), got %d", len(args))
	}
	userID, err := parseID(args[0], "user")
	if err != nil {
		return nil, err
	}

	balances, err := e.svc.GetUserData(ctx, userID)
	if err != nil {
		return nil, err
	}
	return formatBalances(balances), nil
}

// Show_All_User_Balance
func (e *executor) showAllBalances(ctx context.Context, args []string) ([]string, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("Show_All_User_Balance takes no arguments, got %d", len(args))
	}

	balances, err := e.svc.GetAllBalances(ctx)
	if err != nil {
		return nil, err
	}
	return formatBalances(balances), nil
}

// Show_User_Expense [UserId]
func (e *executor) showUserExpense(ctx context.Context, args []string) ([]string, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Show_User_Expense expects 1 argument (userId), got %d", len(args))
	}
	userID, err := parseID(args[0], "user")
	if err != nil {
		return nil, err
	}

	items, err := e.svc.GetUserExpenses(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return []string{"No expenses"}, nil
	}

	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, strings.Join([]string{
			it.Expense.Name,
			string(it.Expense.SplitType),
			"total " + paisaToRupees(it.Expense.AmountPaisa),
			"owed " + paisaToRupees(it.UserOwedPaisa),
			"paid " + paisaToRupees(it.UserPaidPaisa),
		}, " "))
	}
	return lines, nil
}

// ── small parse helpers ──────────────────────────────────────────────────────

func parseID(s, label string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.Errorf("invalid %s id: %q", label, s)
	}
	return id, nil
}

func parseSplitType(s string) (models.SplitType, error) {
	switch strings.ToLower(s) {
	case "equal":
		return models.SplitEqual, nil
	case "exact":
		return models.SplitExact, nil
	case "percent":
		return models.SplitPercent, nil
	default:
		return "", errors.Errorf("unknown split type: %q (want Equal, Exact or Percent)", s)
	}
}

// parseShares reads the type-specific tail: none for equal, exact amounts (paisa)
// for exact, percent basis points for percent. It enforces that the number of
// shares matches the number of participants.
func parseShares(t models.SplitType, tokens []string, n int) ([]int64, error) {
	switch t {
	case models.SplitEqual:
		if len(tokens) != 0 {
			return nil, errors.Errorf("Equal split takes no share values, got %d", len(tokens))
		}
		return nil, nil

	case models.SplitExact:
		if len(tokens) != n {
			return nil, errors.Errorf("Exact split expects %d amounts, got %d", n, len(tokens))
		}
		shares := make([]int64, n)
		for i, tok := range tokens {
			p, err := rupeesToPaisa(tok)
			if err != nil {
				return nil, err
			}
			shares[i] = p
		}
		return shares, nil

	case models.SplitPercent:
		if len(tokens) != n {
			return nil, errors.Errorf("Percent split expects %d percents, got %d", n, len(tokens))
		}
		shares := make([]int64, n)
		for i, tok := range tokens {
			bp, err := percentToBasisPoints(tok)
			if err != nil {
				return nil, err
			}
			shares[i] = bp
		}
		return shares, nil

	default:
		return nil, errors.Errorf("unknown split type: %q", t)
	}
}

// formatBalances renders settlement balances as "From owes To Amount" lines, or
// a single "No balances" line when empty.
func formatBalances(balances []expensesharing.Balance) []string {
	if len(balances) == 0 {
		return []string{"No balances"}
	}
	lines := make([]string, 0, len(balances))
	for _, b := range balances {
		lines = append(lines, strings.Join([]string{
			b.FromUserName, "owes", b.ToUserName, paisaToRupees(b.AmountPaisa),
		}, " "))
	}
	return lines
}
