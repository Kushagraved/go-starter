package main

import (
	"context"

	"go-starter/internal/core/expensesharing"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("executor", func() {
	var (
		ctrl *gomock.Controller
		svc  *expensesharing.MockService
		exec *executor
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		svc = expensesharing.NewMockService(ctrl)
		exec = &executor{svc: svc}
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	Describe("no-op lines", func() {
		It("ignores blank lines and comments", func() {
			out, err := exec.Run(ctx, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeNil())

			out, err = exec.Run(ctx, "   # a comment")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeNil())
		})
	})

	Describe("Add_User", func() {
		It("parses and prints the new user id", func() {
			svc.EXPECT().
				AddUser(gomock.Any(), expensesharing.AddUserRequest{
					Name: "Alice", Email: "alice@example.com", Phone: "9876543210",
				}).
				Return(models.User{ID: 7}, nil)

			out, err := exec.Run(ctx, "Add_User Alice alice@example.com 9876543210")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"7"}))
		})

		It("rejects the wrong argument count", func() {
			out, err := exec.Run(ctx, "Add_User Alice alice@example.com")
			Expect(err).To(HaveOccurred())
			Expect(out).To(BeNil())
		})
	})

	Describe("Add_Expense — Equal", func() {
		It("builds an equal-split request with rupee→paisa conversion", func() {
			svc.EXPECT().
				AddExpense(gomock.Any(), expensesharing.AddExpenseRequest{
					Name:            "Dinner",
					TotalPaisa:      10000,
					PaidByUserID:    1,
					CreatedByUserID: 1,
					SplitType:       models.SplitEqual,
					Participants:    []int64{1, 2, 3},
					Shares:          nil,
				}).
				Return(models.Expense{ID: 5}, nil)

			out, err := exec.Run(ctx, "Add_Expense Dinner 100 1 1 3 Equal 1 2 3")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"expense 5 recorded"}))
		})

		It("rejects extra share values on an equal split", func() {
			_, err := exec.Run(ctx, "Add_Expense Dinner 100 1 1 3 Equal 1 2 3 50")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Add_Expense — Exact", func() {
		It("parses exact amounts into paisa shares", func() {
			svc.EXPECT().
				AddExpense(gomock.Any(), expensesharing.AddExpenseRequest{
					Name:            "Cab",
					TotalPaisa:      10000,
					PaidByUserID:    1,
					CreatedByUserID: 1,
					SplitType:       models.SplitExact,
					Participants:    []int64{1, 2},
					Shares:          []int64{6000, 4000},
				}).
				Return(models.Expense{ID: 6}, nil)

			out, err := exec.Run(ctx, "Add_Expense Cab 100 1 1 2 Exact 1 2 60 40")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"expense 6 recorded"}))
		})

		It("rejects a share count that does not match participants", func() {
			_, err := exec.Run(ctx, "Add_Expense Cab 100 1 1 2 Exact 1 2 60")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Add_Expense — Percent", func() {
		It("parses percents into basis points", func() {
			svc.EXPECT().
				AddExpense(gomock.Any(), expensesharing.AddExpenseRequest{
					Name:            "Groceries",
					TotalPaisa:      10000,
					PaidByUserID:    1,
					CreatedByUserID: 1,
					SplitType:       models.SplitPercent,
					Participants:    []int64{1, 2, 3},
					Shares:          []int64{3333, 3333, 3334},
				}).
				Return(models.Expense{ID: 8}, nil)

			out, err := exec.Run(ctx, "Add_Expense Groceries 100 1 1 3 Percent 1 2 3 33.33 33.33 33.34")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"expense 8 recorded"}))
		})
	})

	Describe("Show_All_User_Balance", func() {
		It("formats each balance as 'From owes To Amount'", func() {
			svc.EXPECT().GetAllBalances(gomock.Any()).Return([]expensesharing.Balance{
				{FromUserName: "User2", ToUserName: "User1", AmountPaisa: 3333},
				{FromUserName: "User3", ToUserName: "User1", AmountPaisa: 3333},
			}, nil)

			out, err := exec.Run(ctx, "Show_All_User_Balance")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{
				"User2 owes User1 33.33",
				"User3 owes User1 33.33",
			}))
		})

		It("prints 'No balances' when there are none", func() {
			svc.EXPECT().GetAllBalances(gomock.Any()).Return(nil, nil)
			out, err := exec.Run(ctx, "Show_All_User_Balance")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"No balances"}))
		})

		It("rejects extra arguments", func() {
			_, err := exec.Run(ctx, "Show_All_User_Balance 1")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Show_User_Data", func() {
		It("prints only that user's balances", func() {
			svc.EXPECT().GetUserData(gomock.Any(), int64(1)).Return([]expensesharing.Balance{
				{FromUserName: "User2", ToUserName: "User1", AmountPaisa: 5000},
			}, nil)

			out, err := exec.Run(ctx, "Show_User_Data 1")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"User2 owes User1 50.00"}))
		})
	})

	Describe("Show_User_Expense", func() {
		It("prints each expense with the user's owed and paid", func() {
			svc.EXPECT().GetUserExpenses(gomock.Any(), int64(1)).Return([]expensesharing.UserExpense{
				{
					Expense:       models.Expense{Name: "Dinner", AmountPaisa: 10000, SplitType: models.SplitEqual},
					UserOwedPaisa: 3334,
					UserPaidPaisa: 10000,
				},
			}, nil)

			out, err := exec.Run(ctx, "Show_User_Expense 1")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"Dinner EQUAL total 100.00 owed 33.34 paid 100.00"}))
		})

		It("prints 'No expenses' when the user has none", func() {
			svc.EXPECT().GetUserExpenses(gomock.Any(), int64(1)).Return(nil, nil)
			out, err := exec.Run(ctx, "Show_User_Expense 1")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal([]string{"No expenses"}))
		})
	})

	Describe("unknown command", func() {
		It("returns an error", func() {
			_, err := exec.Run(ctx, "Frobnicate 1 2 3")
			Expect(err).To(HaveOccurred())
		})
	})
})
