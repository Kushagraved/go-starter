package expensesharing_test

import (
	"context"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/core/expensesharing"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

// These are the adversarial cases: awkward remainders, large amounts, payer not
// first, zero shares, off-by-one sums. The invariant under test throughout is
// that owed always sums to exactly the total and paid is fronted only by the
// payer.
var _ = Describe("AddExpense — hard cases", func() {
	var (
		ctrl *gomock.Controller
		repo *internal.MockRepository
		svc  expensesharing.Service
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = internal.NewMockRepository(ctrl)
		svc = expensesharing.NewService(repo, newTestLogger())
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	expectUsersExist := func(ids ...int64) {
		for _, id := range ids {
			repo.EXPECT().GetUserByID(gomock.Any(), id).Return(models.User{ID: id}, nil).AnyTimes()
		}
	}

	// captureSplits stubs the persist call and hands back the splits it was given.
	captureSplits := func(dst *[]models.Split) {
		repo.EXPECT().
			CreateExpenseWithSplits(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, exp models.Expense, splits []models.Split) (models.Expense, error) {
				*dst = splits
				return exp, nil
			})
	}

	sumOwed := func(splits []models.Split) int64 {
		var t int64
		for _, s := range splits {
			t += s.OwedPaisa
		}
		return t
	}

	sumPaid := func(splits []models.Split) int64 {
		var t int64
		for _, s := range splits {
			t += s.PaidPaisa
		}
		return t
	}

	ids := func(n int) []int64 {
		out := make([]int64, n)
		for i := range out {
			out[i] = int64(i + 1)
		}
		return out
	}

	Describe("equal split — remainder distribution", func() {
		DescribeTable("owed always sums to total; first `remainder` users get +1 paisa",
			func(totalPaisa int64, n int) {
				participants := ids(n)
				expectUsersExist(participants...)
				var got []models.Split
				captureSplits(&got)

				_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
					Name: "X", TotalPaisa: totalPaisa, PaidByUserID: 1, CreatedByUserID: 1,
					SplitType: models.SplitEqual, Participants: participants,
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(sumOwed(got)).To(Equal(totalPaisa), "owed must reconcile to total")

				base := totalPaisa / int64(n)
				rem := totalPaisa % int64(n)
				for i, s := range got {
					want := base
					if int64(i) < rem {
						want++
					}
					Expect(s.OwedPaisa).To(Equal(want), "user index %d", i)
				}
			},
			Entry("100.00 over 3 (spec example)", int64(10000), 3),
			Entry("100.01 over 7 (rem 6 → six users +1)", int64(10001), 7),
			Entry("1 paisa over 5 (only first gets it)", int64(1), 5),
			Entry("99999 over 4 (rem 3)", int64(99999), 4),
			Entry("everyone equal, no remainder", int64(9000), 3),
			Entry("single participant takes all", int64(12345), 1),
			Entry("large amount, prime user count", int64(1_000_000_007), 13),
		)

		It("puts 1 paisa on the first user when total is 1 paisa over many users", func() {
			participants := ids(5)
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 1, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitEqual, Participants: participants,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(got[0].OwedPaisa).To(Equal(int64(1)))
			for i := 1; i < 5; i++ {
				Expect(got[i].OwedPaisa).To(Equal(int64(0)))
			}
		})
	})

	Describe("payer placement", func() {
		It("fronts the full amount on the payer's row even when the payer is not first", func() {
			participants := []int64{1, 2, 3}
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			// Payer is user 3 (last), creator is user 2 (middle, also a participant).
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 3, CreatedByUserID: 2,
				SplitType: models.SplitEqual, Participants: participants,
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(got[0].PaidPaisa).To(Equal(int64(0)))
			Expect(got[1].PaidPaisa).To(Equal(int64(0)))
			Expect(got[2].PaidPaisa).To(Equal(int64(10000)))
			Expect(sumPaid(got)).To(Equal(int64(10000)), "exactly the total is fronted")

			// Rounding leftover still lands on the first participant, independent
			// of who paid.
			Expect(got[0].OwedPaisa).To(Equal(int64(3334)))
		})

		It("allows a creator who is not a participant, as long as the payer is", func() {
			participants := []int64{1, 2}
			expectUsersExist(1, 2, 99) // creator 99 exists but is not in the split
			var got []models.Split
			captureSplits(&got)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 5000, PaidByUserID: 1, CreatedByUserID: 99,
				SplitType: models.SplitEqual, Participants: participants,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(sumPaid(got)).To(Equal(int64(5000)))
		})
	})

	Describe("exact split — boundaries", func() {
		It("accepts a zero share as long as the rest sum to the total", func() {
			participants := []int64{1, 2, 3}
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitExact, Participants: participants,
				Shares: []int64{10000, 0, 0},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(got[1].OwedPaisa).To(Equal(int64(0)))
			Expect(sumOwed(got)).To(Equal(int64(10000)))
		})

		It("rejects shares that overshoot the total by exactly 1 paisa", func() {
			expectUsersExist(1, 2)
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitExact, Participants: []int64{1, 2},
				Shares: []int64{5000, 5001}, // 10001
			})
			Expect(err).To(MatchError(constants.ErrExactSumMismatch))
		})

		It("rejects shares that undershoot the total by exactly 1 paisa", func() {
			expectUsersExist(1, 2)
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitExact, Participants: []int64{1, 2},
				Shares: []int64{5000, 4999}, // 9999
			})
			Expect(err).To(MatchError(constants.ErrExactSumMismatch))
		})

		It("rejects a negative share", func() {
			expectUsersExist(1, 2)
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitExact, Participants: []int64{1, 2},
				Shares: []int64{11000, -1000},
			})
			Expect(err).To(MatchError(constants.ErrNegativeShare))
		})

		It("rejects when the number of shares does not match participants", func() {
			expectUsersExist(1, 2, 3)
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitExact, Participants: []int64{1, 2, 3},
				Shares: []int64{5000, 5000}, // only 2 shares for 3 users
			})
			Expect(err).To(MatchError(constants.ErrShareCountMismatch))
		})
	})

	Describe("percent split — maximum floor loss", func() {
		It("never leaks paisa: leftover distributed stays within participant count", func() {
			// Seven users, awkward split that maximises floor loss. Percents must
			// sum to exactly 10000 bp. Use 1429 x6 + 1426 = 10000.
			participants := ids(7)
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			shares := []int64{1429, 1429, 1429, 1429, 1429, 1429, 1426}
			var bp int64
			for _, s := range shares {
				bp += s
			}
			Expect(bp).To(Equal(int64(10000)), "test data must sum to 100%")

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitPercent, Participants: participants, Shares: shares,
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(sumOwed(got)).To(Equal(int64(10000)), "owed must reconcile to total")
		})

		It("accepts a 0% participant and still reconciles", func() {
			participants := []int64{1, 2, 3}
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitPercent, Participants: participants,
				Shares: []int64{5000, 5000, 0}, // 50/50/0
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(got[2].OwedPaisa).To(Equal(int64(0)))
			Expect(sumOwed(got)).To(Equal(int64(10000)))
		})

		It("handles a large total without overflow and reconciles exactly", func() {
			participants := ids(3)
			expectUsersExist(participants...)
			var got []models.Split
			captureSplits(&got)

			// ₹10,000,000.00 = 1_000_000_000 paisa, split 33.33/33.33/33.34.
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 1_000_000_000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitPercent, Participants: participants,
				Shares: []int64{3333, 3333, 3334},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(sumOwed(got)).To(Equal(int64(1_000_000_000)))
		})

		It("rejects percents that sum to 100.01 (off by 1 bp)", func() {
			expectUsersExist(1, 2)
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitPercent, Participants: []int64{1, 2},
				Shares: []int64{5000, 5001}, // 100.01%
			})
			Expect(err).To(MatchError(constants.ErrPercentSumMismatch))
		})
	})

	Describe("validation precedence", func() {
		It("reports the bad total before touching the repository", func() {
			// No GetUserByID / CreateExpenseWithSplits expectations set: if the
			// service called them, the mock controller would fail the test.
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: -1, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitEqual, Participants: []int64{1, 2},
			})
			Expect(err).To(MatchError(constants.ErrNonPositiveAmount))
		})

		It("reports a duplicate participant before checking existence", func() {
			// Again, no repo expectations: duplicate detection must short-circuit.
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name: "X", TotalPaisa: 10000, PaidByUserID: 1, CreatedByUserID: 1,
				SplitType: models.SplitEqual, Participants: []int64{1, 2, 2},
			})
			Expect(err).To(MatchError(constants.ErrDuplicateUserInSplit))
		})
	})
})
