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

var _ = Describe("AddExpense", func() {
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

	// expectUsersExist stubs GetUserByID to succeed for the given ids (any order,
	// any number of calls per id).
	expectUsersExist := func(ids ...int64) {
		for _, id := range ids {
			repo.EXPECT().
				GetUserByID(gomock.Any(), id).
				Return(models.User{ID: id}, nil).
				AnyTimes()
		}
	}

	Describe("equal split — the spec's rounding example", func() {
		It("assigns 33.34 to the first user and 33.33 to the rest, payer paid full", func() {
			expectUsersExist(1, 2, 3)

			var captured []models.Split
			repo.EXPECT().
				CreateExpenseWithSplits(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, exp models.Expense, splits []models.Split) (models.Expense, error) {
					captured = splits
					exp.ID = 1
					exp.Splits = splits
					return exp, nil
				})

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Dinner",
				TotalPaisa:      10000, // ₹100.00
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitEqual,
				Participants:    []int64{1, 2, 3},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(captured).To(HaveLen(3))
			Expect(captured[0].OwedPaisa).To(Equal(int64(3334))) // first absorbs remainder
			Expect(captured[1].OwedPaisa).To(Equal(int64(3333)))
			Expect(captured[2].OwedPaisa).To(Equal(int64(3333)))

			// Owed sums back to the total.
			Expect(captured[0].OwedPaisa + captured[1].OwedPaisa + captured[2].OwedPaisa).
				To(Equal(int64(10000)))

			// Payer fronted the full amount; others fronted nothing.
			Expect(captured[0].PaidPaisa).To(Equal(int64(10000)))
			Expect(captured[1].PaidPaisa).To(Equal(int64(0)))
			Expect(captured[2].PaidPaisa).To(Equal(int64(0)))
		})
	})

	Describe("exact split", func() {
		It("uses the provided amounts when they sum to the total", func() {
			expectUsersExist(1, 2)

			var captured []models.Split
			repo.EXPECT().
				CreateExpenseWithSplits(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, exp models.Expense, splits []models.Split) (models.Expense, error) {
					captured = splits
					return exp, nil
				})

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Cab",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitExact,
				Participants:    []int64{1, 2},
				Shares:          []int64{6000, 4000},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(captured[0].OwedPaisa).To(Equal(int64(6000)))
			Expect(captured[1].OwedPaisa).To(Equal(int64(4000)))
		})

		It("rejects when the exact shares do not sum to the total", func() {
			expectUsersExist(1, 2)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Cab",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitExact,
				Participants:    []int64{1, 2},
				Shares:          []int64{6000, 3000}, // 9000 != 10000
			})
			Expect(err).To(MatchError(constants.ErrExactSumMismatch))
		})
	})

	Describe("percent split", func() {
		It("converts percents to paisa and pushes the leftover to the first user", func() {
			expectUsersExist(1, 2, 3)

			var captured []models.Split
			repo.EXPECT().
				CreateExpenseWithSplits(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, exp models.Expense, splits []models.Split) (models.Expense, error) {
					captured = splits
					return exp, nil
				})

			// 100.00 total, split 33.33% / 33.33% / 33.34% (basis points).
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Groceries",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitPercent,
				Participants:    []int64{1, 2, 3},
				Shares:          []int64{3333, 3333, 3334},
			})

			Expect(err).ToNot(HaveOccurred())
			// floor gives 3333/3333/3334 = 10000 exactly, no leftover here.
			Expect(captured[0].OwedPaisa + captured[1].OwedPaisa + captured[2].OwedPaisa).
				To(Equal(int64(10000)))
		})

		It("reconciles floor loss onto the first user so owed sums to the total", func() {
			expectUsersExist(1, 2, 3)

			var captured []models.Split
			repo.EXPECT().
				CreateExpenseWithSplits(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, exp models.Expense, splits []models.Split) (models.Expense, error) {
					captured = splits
					return exp, nil
				})

			// ₹1.00 = 100 paisa split 33.33/33.33/33.34%. Each floor is 33 paisa
			// (99 total), leaving 1 paisa of loss that must land on the first user.
			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Snacks",
				TotalPaisa:      100, // ₹1.00
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitPercent,
				Participants:    []int64{1, 2, 3},
				Shares:          []int64{3333, 3333, 3334}, // floors: 33,33,33 -> leftover 1
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(captured[0].OwedPaisa + captured[1].OwedPaisa + captured[2].OwedPaisa).
				To(Equal(int64(100)))
			Expect(captured[0].OwedPaisa).To(Equal(int64(34))) // first absorbs the leftover paisa
			Expect(captured[1].OwedPaisa).To(Equal(int64(33)))
			Expect(captured[2].OwedPaisa).To(Equal(int64(33)))
		})

		It("rejects when the percent shares do not sum to 100", func() {
			expectUsersExist(1, 2)

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Groceries",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitPercent,
				Participants:    []int64{1, 2},
				Shares:          []int64{6000, 3000}, // 90.00% != 100
			})
			Expect(err).To(MatchError(constants.ErrPercentSumMismatch))
		})
	})

	Describe("structural validation", func() {
		base := func() expensesharing.AddExpenseRequest {
			return expensesharing.AddExpenseRequest{
				Name:            "Dinner",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitEqual,
				Participants:    []int64{1, 2},
			}
		}

		It("rejects an empty name", func() {
			req := base()
			req.Name = ""
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrEmptyExpenseName))
		})

		It("rejects a non-positive total", func() {
			req := base()
			req.TotalPaisa = 0
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrNonPositiveAmount))
		})

		It("rejects an empty participant list", func() {
			req := base()
			req.Participants = nil
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrEmptyUserList))
		})

		It("rejects a duplicate participant", func() {
			req := base()
			req.Participants = []int64{1, 1}
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrDuplicateUserInSplit))
		})

		It("rejects when the payer is not a participant", func() {
			req := base()
			req.PaidByUserID = 99
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrPayerNotInSplit))
		})

		It("rejects an unknown split type", func() {
			req := base()
			req.SplitType = models.SplitType("WEIRD")
			expectUsersExist(1, 2)
			_, err := svc.AddExpense(ctx, req)
			Expect(err).To(MatchError(constants.ErrInvalidSplitType))
		})
	})

	Describe("user existence", func() {
		It("returns RecordDoesntExists when a participant is missing", func() {
			// User 1 exists, user 2 does not.
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil).AnyTimes()
			repo.EXPECT().GetUserByID(gomock.Any(), int64(2)).Return(models.User{}, constants.RecordDoesntExists).AnyTimes()

			_, err := svc.AddExpense(ctx, expensesharing.AddExpenseRequest{
				Name:            "Dinner",
				TotalPaisa:      10000,
				PaidByUserID:    1,
				CreatedByUserID: 1,
				SplitType:       models.SplitEqual,
				Participants:    []int64{1, 2},
			})
			Expect(err).To(MatchError(constants.RecordDoesntExists))
		})
	})
})
