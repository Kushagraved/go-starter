package expensesharing_test

import (
	"context"
	"errors"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/core/expensesharing"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("GetUserExpenses", func() {
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

	Describe("happy path", func() {
		It("returns each expense with the target user's owed and paid amounts", func() {
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1, Name: "User1"}, nil)

			// Expense 10: user 1 is the payer (paid 10000, owed 3334).
			// Expense 20: user 1 only owes a share (paid 0, owed 2000).
			repo.EXPECT().ListExpensesByUserID(gomock.Any(), int64(1)).Return([]models.Expense{
				{
					ID: 10, Name: "Dinner", AmountPaisa: 10000, SplitType: models.SplitEqual,
					Splits: []models.Split{
						{ExpenseID: 10, UserID: 1, OwedPaisa: 3334, PaidPaisa: 10000},
						{ExpenseID: 10, UserID: 2, OwedPaisa: 3333, PaidPaisa: 0},
						{ExpenseID: 10, UserID: 3, OwedPaisa: 3333, PaidPaisa: 0},
					},
				},
				{
					ID: 20, Name: "Cab", AmountPaisa: 4000, SplitType: models.SplitExact,
					Splits: []models.Split{
						{ExpenseID: 20, UserID: 1, OwedPaisa: 2000, PaidPaisa: 0},
						{ExpenseID: 20, UserID: 4, OwedPaisa: 2000, PaidPaisa: 4000},
					},
				},
			}, nil)

			result, err := svc.GetUserExpenses(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(2))

			Expect(result[0].Expense.Name).To(Equal("Dinner"))
			Expect(result[0].UserOwedPaisa).To(Equal(int64(3334)))
			Expect(result[0].UserPaidPaisa).To(Equal(int64(10000)))

			Expect(result[1].Expense.Name).To(Equal("Cab"))
			Expect(result[1].UserOwedPaisa).To(Equal(int64(2000)))
			Expect(result[1].UserPaidPaisa).To(Equal(int64(0)))
		})

		It("returns an empty list for a user who exists but has no expenses", func() {
			repo.EXPECT().GetUserByID(gomock.Any(), int64(7)).Return(models.User{ID: 7}, nil)
			repo.EXPECT().ListExpensesByUserID(gomock.Any(), int64(7)).Return([]models.Expense{}, nil)

			result, err := svc.GetUserExpenses(ctx, 7)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})

	Describe("unknown user", func() {
		It("returns RecordDoesntExists and never lists expenses", func() {
			// No ListExpensesByUserID expectation: existence check must short-circuit.
			repo.EXPECT().GetUserByID(gomock.Any(), int64(99)).
				Return(models.User{}, constants.RecordDoesntExists)

			_, err := svc.GetUserExpenses(ctx, 99)
			Expect(err).To(MatchError(constants.RecordDoesntExists))
		})
	})

	Describe("repository errors", func() {
		It("propagates a list error", func() {
			boom := errors.New("db down")
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil)
			repo.EXPECT().ListExpensesByUserID(gomock.Any(), int64(1)).Return(nil, boom)

			_, err := svc.GetUserExpenses(ctx, 1)
			Expect(err).To(MatchError(boom))
		})
	})

	Describe("edge case", func() {
		It("reports zero owed/paid if the user is somehow absent from an expense's splits", func() {
			// Defensive: the repository returned an expense with no split for the
			// user (should not happen, but the shaping must not panic or leak).
			repo.EXPECT().GetUserByID(gomock.Any(), int64(1)).Return(models.User{ID: 1}, nil)
			repo.EXPECT().ListExpensesByUserID(gomock.Any(), int64(1)).Return([]models.Expense{
				{
					ID: 30, Name: "Orphan", AmountPaisa: 5000, SplitType: models.SplitEqual,
					Splits: []models.Split{
						{ExpenseID: 30, UserID: 2, OwedPaisa: 5000, PaidPaisa: 5000},
					},
				},
			}, nil)

			result, err := svc.GetUserExpenses(ctx, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].UserOwedPaisa).To(Equal(int64(0)))
			Expect(result[0].UserPaidPaisa).To(Equal(int64(0)))
		})
	})
})
