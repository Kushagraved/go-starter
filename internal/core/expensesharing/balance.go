package expensesharing

import (
	"container/heap"
	"context"
	"sort"

	"go-starter/internal/logger"
	"go-starter/internal/models"
)

// GetAllBalances derives every non-zero balance across all expenses as a minimum
// settlement.
//
// Balances are computed on read from the raw splits — there is no stored balance
// table — so they can never drift out of sync with the expenses.
func (s *service) GetAllBalances(ctx context.Context) ([]Balance, error) {
	splits, err := s.repo.ListAllSplits(ctx)
	if err != nil {
		s.logger.Error(ctx, "ExpenseSharing", "GetAllBalances", "failed to list splits", logger.NewError(err))
		return nil, err
	}

	users, err := s.repo.ListUsers(ctx)
	if err != nil {
		s.logger.Error(ctx, "ExpenseSharing", "GetAllBalances", "failed to list users", logger.NewError(err))
		return nil, err
	}

	names := make(map[int64]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Name
	}

	return settleBalances(splits, names), nil
}

// GetUserData returns the settlement balances that involve the given user, on
// either side. The user must exist.
func (s *service) GetUserData(ctx context.Context, userID int64) ([]Balance, error) {
	if _, err := s.repo.GetUserByID(ctx, userID); err != nil {
		return nil, err
	}

	all, err := s.GetAllBalances(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]Balance, 0)
	for _, b := range all {
		if b.FromUserID == userID || b.ToUserID == userID {
			out = append(out, b)
		}
	}
	return out, nil
}

// settleBalances computes the minimum set of transfers that settles everyone,
// using two priority queues.
//
// It works at the global level, not per expense: each user's net across ALL
// their splits is paid - owed. A positive net is a creditor (the group owes
// them), a negative net is a debtor (they owe the group); the nets sum to zero.
//
// The largest creditor and the largest debtor are always matched first:
//   - creditors go into a max-heap keyed by amount owed to them,
//   - debtors go into a max-heap keyed by amount they owe.
//
// Pop the top of each, settle min(debt, credit), and push back whichever side
// still has a remainder. Repeat until a queue drains. Matching the two extremes
// keeps the number of transfers small.
//
// Consequence to be aware of: this can produce a transfer between two users who
// never shared an expense directly (an intermediary who nets to zero drops out).
// That is the intended "minimum settlement" behaviour.
func settleBalances(splits []models.Split, names map[int64]string) []Balance {
	// 1. Net per user across every split.
	netByUser := make(map[int64]int64)
	for _, sp := range splits {
		netByUser[sp.UserID] += sp.PaidPaisa - sp.OwedPaisa
	}

	// 2. Seed the two heaps. Both are max-heaps on positive magnitude; user id is
	// the tiebreaker so the output is deterministic.
	creditors := &partyHeap{}
	debtors := &partyHeap{}
	heap.Init(creditors)
	heap.Init(debtors)

	// Iterate user ids in order so equal amounts break ties deterministically.
	ids := make([]int64, 0, len(netByUser))
	for id := range netByUser {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		net := netByUser[id]
		switch {
		case net > 0:
			heap.Push(creditors, party{userID: id, amount: net})
		case net < 0:
			heap.Push(debtors, party{userID: id, amount: -net}) // store positive magnitude
		}
	}

	// 3. Match the biggest debtor with the biggest creditor until settled.
	var balances []Balance
	for creditors.Len() > 0 && debtors.Len() > 0 {
		cred := heap.Pop(creditors).(party)
		debt := heap.Pop(debtors).(party)

		settled := cred.amount
		if debt.amount < settled {
			settled = debt.amount // min(|debt|, |credit|)
		}

		balances = append(balances, Balance{
			FromUserID:   debt.userID,
			FromUserName: names[debt.userID],
			ToUserID:     cred.userID,
			ToUserName:   names[cred.userID],
			AmountPaisa:  settled,
		})

		// Push back whichever side still has money left.
		if rem := cred.amount - settled; rem > 0 {
			heap.Push(creditors, party{userID: cred.userID, amount: rem})
		}
		if rem := debt.amount - settled; rem > 0 {
			heap.Push(debtors, party{userID: debt.userID, amount: rem})
		}
	}

	// 4. Stable output ordering for callers/tests.
	sort.Slice(balances, func(i, j int) bool {
		if balances[i].FromUserID != balances[j].FromUserID {
			return balances[i].FromUserID < balances[j].FromUserID
		}
		return balances[i].ToUserID < balances[j].ToUserID
	})
	return balances
}

// party is one user's outstanding magnitude in a heap (either credit or debt).
type party struct {
	userID int64
	amount int64 // always positive
}

// partyHeap is a max-heap on amount, with user id as a deterministic tiebreaker.
// It implements heap.Interface. The same type serves both the creditor and the
// debtor queues because both are "largest magnitude first".
type partyHeap []party

func (h partyHeap) Len() int { return len(h) }

func (h partyHeap) Less(i, j int) bool {
	if h[i].amount != h[j].amount {
		return h[i].amount > h[j].amount // max-heap: larger amount is "less" (root)
	}
	return h[i].userID < h[j].userID
}

func (h partyHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *partyHeap) Push(x any) { *h = append(*h, x.(party)) }

func (h *partyHeap) Pop() any {
	old := *h
	n := len(old)
	top := old[n-1]
	*h = old[:n-1]
	return top
}
