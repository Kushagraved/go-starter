package expensesharing_test

import (
	"context"
	"errors"

	"go-starter/internal"
	"go-starter/internal/core/expensesharing"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("GetAllBalances (global settlement)", func() {
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

	stubUsers := func(ids ...int64) {
		users := make([]models.User, 0, len(ids))
		for _, id := range ids {
			users = append(users, models.User{ID: id, Name: nameFor(id)})
		}
		repo.EXPECT().ListUsers(gomock.Any()).Return(users, nil)
	}

	findBalance := func(bs []expensesharing.Balance, from, to int64) *expensesharing.Balance {
		for i := range bs {
			if bs[i].FromUserID == from && bs[i].ToUserID == to {
				return &bs[i]
			}
		}
		return nil
	}

	sumAmounts := func(bs []expensesharing.Balance) int64 {
		var t int64
		for _, b := range bs {
			t += b.AmountPaisa
		}
		return t
	}

	Describe("the spec example", func() {
		It("U1 pays 100 split equally 3 ways: U2 and U3 each owe U1 33.33", func() {
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				{ExpenseID: 10, UserID: 1, OwedPaisa: 3334, PaidPaisa: 10000},
				{ExpenseID: 10, UserID: 2, OwedPaisa: 3333, PaidPaisa: 0},
				{ExpenseID: 10, UserID: 3, OwedPaisa: 3333, PaidPaisa: 0},
			}, nil)
			stubUsers(1, 2, 3)

			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(balances).To(HaveLen(2))

			b2 := findBalance(balances, 2, 1)
			Expect(b2).ToNot(BeNil())
			Expect(b2.AmountPaisa).To(Equal(int64(3333)))
			Expect(b2.FromUserName).To(Equal("User2"))
			Expect(b2.ToUserName).To(Equal("User1"))

			b3 := findBalance(balances, 3, 1)
			Expect(b3).ToNot(BeNil())
			Expect(b3.AmountPaisa).To(Equal(int64(3333)))
		})
	})

	Describe("global netting across expenses", func() {
		It("nets a user's opposing positions into a single balance", func() {
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				// Expense 1: U1 pays 10000, split 50/50 → U2 owes U1 5000.
				{ExpenseID: 1, UserID: 1, OwedPaisa: 5000, PaidPaisa: 10000},
				{ExpenseID: 1, UserID: 2, OwedPaisa: 5000, PaidPaisa: 0},
				// Expense 2: U2 pays 6000, split 50/50 → U1 owes U2 3000.
				{ExpenseID: 2, UserID: 1, OwedPaisa: 3000, PaidPaisa: 0},
				{ExpenseID: 2, UserID: 2, OwedPaisa: 3000, PaidPaisa: 6000},
			}, nil)
			stubUsers(1, 2)

			// Nets: U1 = +5000 - 3000 = +2000 (creditor);
			//       U2 = -5000 + 3000 = -2000 (debtor). So U2 owes U1 2000.
			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(balances).To(HaveLen(1))
			Expect(balances[0].FromUserID).To(Equal(int64(2)))
			Expect(balances[0].ToUserID).To(Equal(int64(1)))
			Expect(balances[0].AmountPaisa).To(Equal(int64(2000)))
		})

		It("omits users whose net is exactly zero", func() {
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				{ExpenseID: 1, UserID: 1, OwedPaisa: 5000, PaidPaisa: 10000},
				{ExpenseID: 1, UserID: 2, OwedPaisa: 5000, PaidPaisa: 0},
				{ExpenseID: 2, UserID: 1, OwedPaisa: 5000, PaidPaisa: 0},
				{ExpenseID: 2, UserID: 2, OwedPaisa: 5000, PaidPaisa: 10000},
			}, nil)
			stubUsers(1, 2)

			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(balances).To(BeEmpty())
		})
	})

	Describe("intermediary drops out (the defining property of global settlement)", func() {
		It("routes a debt around a pass-through user who nets to zero", func() {
			// Expense 1: U1 pays 10000 split with U2 → U2 owes 5000, U1 net +5000.
			// Expense 2: U3 pays 10000 split with U1 → U1 owes 5000, so U1 net 0.
			// Globally: U1 = 0, U2 = -5000, U3 = +5000.
			// Result: U2 owes U3 5000, even though they never shared an expense,
			// and U1 vanishes.
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				{ExpenseID: 1, UserID: 1, OwedPaisa: 5000, PaidPaisa: 10000},
				{ExpenseID: 1, UserID: 2, OwedPaisa: 5000, PaidPaisa: 0},
				{ExpenseID: 2, UserID: 1, OwedPaisa: 5000, PaidPaisa: 0},
				{ExpenseID: 2, UserID: 3, OwedPaisa: 5000, PaidPaisa: 10000},
			}, nil)
			stubUsers(1, 2, 3)

			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(balances).To(HaveLen(1))
			Expect(balances[0].FromUserID).To(Equal(int64(2)))
			Expect(balances[0].ToUserID).To(Equal(int64(3)))
			Expect(balances[0].AmountPaisa).To(Equal(int64(5000)))
			Expect(findBalance(balances, 2, 1)).To(BeNil(), "U1 is a pass-through and must not appear")
			Expect(findBalance(balances, 1, 3)).To(BeNil())
		})
	})

	Describe("one debtor split across multiple creditors", func() {
		It("greedily pays the largest creditor first and reconciles to the debt", func() {
			// U3 owes 10000 total; U1 is owed 6000, U2 is owed 4000.
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				{ExpenseID: 1, UserID: 1, OwedPaisa: 0, PaidPaisa: 6000},
				{ExpenseID: 1, UserID: 2, OwedPaisa: 0, PaidPaisa: 4000},
				{ExpenseID: 1, UserID: 3, OwedPaisa: 10000, PaidPaisa: 0},
			}, nil)
			stubUsers(1, 2, 3)

			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())

			b31 := findBalance(balances, 3, 1)
			b32 := findBalance(balances, 3, 2)
			Expect(b31).ToNot(BeNil())
			Expect(b32).ToNot(BeNil())
			Expect(b31.AmountPaisa).To(Equal(int64(6000)))
			Expect(b32.AmountPaisa).To(Equal(int64(4000)))
			Expect(sumAmounts(balances)).To(Equal(int64(10000)), "transfers reconcile to the debt")
		})
	})

	Describe("multiple debtors and creditors", func() {
		It("produces transfers whose totals match each side's net", func() {
			// Nets: U1 +7000 (creditor), U2 +3000 (creditor),
			//       U3 -8000 (debtor),   U4 -2000 (debtor). Sum = 0.
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{
				{ExpenseID: 1, UserID: 1, OwedPaisa: 0, PaidPaisa: 7000},
				{ExpenseID: 1, UserID: 2, OwedPaisa: 0, PaidPaisa: 3000},
				{ExpenseID: 1, UserID: 3, OwedPaisa: 8000, PaidPaisa: 0},
				{ExpenseID: 1, UserID: 4, OwedPaisa: 2000, PaidPaisa: 0},
			}, nil)
			stubUsers(1, 2, 3, 4)

			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())

			// Everything a debtor sends equals their debt; everything a creditor
			// receives equals their credit.
			sentBy := map[int64]int64{}
			recvBy := map[int64]int64{}
			for _, b := range balances {
				sentBy[b.FromUserID] += b.AmountPaisa
				recvBy[b.ToUserID] += b.AmountPaisa
			}
			Expect(sentBy[3]).To(Equal(int64(8000)))
			Expect(sentBy[4]).To(Equal(int64(2000)))
			Expect(recvBy[1]).To(Equal(int64(7000)))
			Expect(recvBy[2]).To(Equal(int64(3000)))
			Expect(sumAmounts(balances)).To(Equal(int64(10000)))
		})
	})

	Describe("empty state", func() {
		It("returns no balances when there are no splits", func() {
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{}, nil)
			stubUsers()
			balances, err := svc.GetAllBalances(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(balances).To(BeEmpty())
		})
	})

	Describe("repository errors", func() {
		It("propagates a splits-listing error", func() {
			boom := errors.New("db down")
			repo.EXPECT().ListAllSplits(gomock.Any()).Return(nil, boom)
			_, err := svc.GetAllBalances(ctx)
			Expect(err).To(MatchError(boom))
		})

		It("propagates a users-listing error", func() {
			boom := errors.New("db down")
			repo.EXPECT().ListAllSplits(gomock.Any()).Return([]models.Split{}, nil)
			repo.EXPECT().ListUsers(gomock.Any()).Return(nil, boom)
			_, err := svc.GetAllBalances(ctx)
			Expect(err).To(MatchError(boom))
		})
	})
})
