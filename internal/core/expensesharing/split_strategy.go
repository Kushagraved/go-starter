package expensesharing

import (
	"go-starter/internal/constants"
	"go-starter/internal/models"
)

// splitStrategy computes how much each participant owes, in paisa, for a given
// total. Each split type (equal, exact, percent) is one strategy. The strategy
// owns its own validation (e.g. exact must sum to the total, percent to 100) and
// the rounding rule: any leftover paisa is assigned to the first participant.
//
// owed is returned index-aligned with the participant list the caller passed in.
type splitStrategy interface {
	// Split returns the owed paisa per participant, or a validation sentinel.
	//   totalPaisa   — the expense total in paisa.
	//   participants — user ids, in order; len is the number of shares.
	//   shares       — type-specific input: nil for equal, exact amounts (paisa)
	//                  for exact, percent-basis-points for percent.
	Split(totalPaisa int64, participants []int64, shares []int64) ([]int64, error)
}

// strategyFor returns the strategy for a split type, or an error for an unknown
// type. This is the single place that maps a SplitType to behaviour.
func strategyFor(t models.SplitType) (splitStrategy, error) {
	switch t {
	case models.SplitEqual:
		return equalSplit{}, nil
	case models.SplitExact:
		return exactSplit{}, nil
	case models.SplitPercent:
		return percentSplit{}, nil
	default:
		return nil, constants.ErrInvalidSplitType
	}
}

// ── Equal ────────────────────────────────────────────────────────────────────

type equalSplit struct{}

// Split divides the total as evenly as possible: base = total/n to everyone, and
// the remainder (total%n) distributed one paisa each to the first `remainder`
// participants. So the first person absorbs the rounding leftover.
//
// Example: 10000 paisa over 3 → base 3333, remainder 1 → [3334, 3333, 3333].
func (equalSplit) Split(totalPaisa int64, participants []int64, _ []int64) ([]int64, error) {
	n := int64(len(participants))
	base := totalPaisa / n
	remainder := totalPaisa % n

	owed := make([]int64, n)
	for i := range owed {
		owed[i] = base
		if int64(i) < remainder {
			owed[i]++
		}
	}
	return owed, nil
}

// ── Exact ──────────────────────────────────────────────────────────────────

type exactSplit struct{}

// Split uses the provided per-user amounts (paisa) directly, after checking they
// sum to the total. Nothing is rounded — the caller stated exact figures.
func (exactSplit) Split(totalPaisa int64, participants []int64, shares []int64) ([]int64, error) {
	if len(shares) != len(participants) {
		return nil, constants.ErrShareCountMismatch
	}

	var sum int64
	for _, s := range shares {
		if s < 0 {
			return nil, constants.ErrNegativeShare
		}
		sum += s
	}
	if sum != totalPaisa {
		return nil, constants.ErrExactSumMismatch
	}

	owed := make([]int64, len(shares))
	copy(owed, shares)
	return owed, nil
}

// ── Percent ────────────────────────────────────────────────────────────────

// percentBasisPointsTotal is 100.00% expressed in basis points (percent × 100),
// so the sum check is exact integer arithmetic with no float rounding.
const percentBasisPointsTotal int64 = 10000

type percentSplit struct{}

// Split validates that the percent shares (in basis points) sum to 100.00%, then
// converts each to paisa. Because percentages of an integer total rarely divide
// evenly, it distributes any leftover paisa one at a time to the first
// participants — the same first-person-absorbs-the-remainder rule as equal.
func (percentSplit) Split(totalPaisa int64, participants []int64, shares []int64) ([]int64, error) {
	if len(shares) != len(participants) {
		return nil, constants.ErrShareCountMismatch
	}

	var sumBp int64
	for _, s := range shares {
		if s < 0 {
			return nil, constants.ErrNegativeShare
		}
		sumBp += s
	}
	if sumBp != percentBasisPointsTotal {
		return nil, constants.ErrPercentSumMismatch
	}

	owed := make([]int64, len(shares))
	var assigned int64
	for i, bp := range shares {
		// floor(total * bp / 10000); leftover reconciled below.
		owed[i] = (totalPaisa * bp) / percentBasisPointsTotal
		assigned += owed[i]
	}

	// Distribute the leftover paisa (total - assigned) to the first participants,
	// one paisa each, so the owed amounts sum back to exactly totalPaisa.
	for i := int64(0); i < totalPaisa-assigned; i++ {
		owed[i]++
	}
	return owed, nil
}
