package onboarding_test

import (
	"context"

	"go-starter/internal"
	"go-starter/internal/constants"
	"go-starter/internal/core/onboarding"
	"go-starter/internal/models"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Onboarding Service", func() {
	var (
		ctrl *gomock.Controller
		repo *internal.MockRepository
		svc  onboarding.Service
		ctx  context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = internal.NewMockRepository(ctrl)
		svc = onboarding.NewService(repo, nil)
		ctx = context.Background()
	})

	AfterEach(func() { ctrl.Finish() })

	Describe("CreateBranch", func() {
		It("creates a branch", func() {
			repo.EXPECT().CreateBranch(gomock.Any(), gomock.Any()).
				Return(models.Branch{ID: 1, Name: "Downtown"}, nil)
			b, err := svc.CreateBranch(ctx, onboarding.CreateBranchRequest{Name: "Downtown"})
			Expect(err).ToNot(HaveOccurred())
			Expect(b.ID).To(Equal(int64(1)))
		})

		It("rejects an empty name", func() {
			_, err := svc.CreateBranch(ctx, onboarding.CreateBranchRequest{Name: ""})
			Expect(err).To(MatchError(constants.ErrValidation))
		})
	})

	Describe("CreateCar", func() {
		It("adds a car after confirming the branch exists", func() {
			repo.EXPECT().GetBranchByID(gomock.Any(), int64(1)).
				Return(models.Branch{ID: 1}, nil)
			repo.EXPECT().CreateCar(gomock.Any(), gomock.Any()).
				Return(models.Car{ID: 10, BranchID: 1, Category: models.CategorySUV}, nil)

			c, err := svc.CreateCar(ctx, onboarding.CreateCarRequest{BranchID: 1, Category: models.CategorySUV})
			Expect(err).ToNot(HaveOccurred())
			Expect(c.ID).To(Equal(int64(10)))
		})

		It("rejects an invalid category before hitting the repo", func() {
			_, err := svc.CreateCar(ctx, onboarding.CreateCarRequest{BranchID: 1, Category: "TRUCK"})
			Expect(err).To(MatchError(constants.ErrValidation))
		})

		It("propagates a missing branch", func() {
			repo.EXPECT().GetBranchByID(gomock.Any(), int64(99)).
				Return(models.Branch{}, constants.RecordDoesntExists)
			_, err := svc.CreateCar(ctx, onboarding.CreateCarRequest{BranchID: 99, Category: models.CategorySedan})
			Expect(err).To(MatchError(constants.RecordDoesntExists))
		})
	})

	Describe("SetPrice", func() {
		It("configures a price for an existing branch", func() {
			repo.EXPECT().GetBranchByID(gomock.Any(), int64(1)).Return(models.Branch{ID: 1}, nil)
			repo.EXPECT().UpsertPriceConfiguration(gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, cfg models.PriceConfiguration) (models.PriceConfiguration, error) {
					Expect(cfg.Unit).To(Equal(models.UnitPerHour))
					cfg.ID = 1
					return cfg, nil
				})

			cfg, err := svc.SetPrice(ctx, onboarding.SetPriceRequest{BranchID: 1, Category: models.CategorySedan, Amount: 500})
			Expect(err).ToNot(HaveOccurred())
			Expect(cfg.Amount).To(Equal(int64(500)))
		})

		DescribeTable("rejects bad pricing input",
			func(req onboarding.SetPriceRequest) {
				_, err := svc.SetPrice(ctx, req)
				Expect(err).To(MatchError(constants.ErrValidation))
			},
			Entry("zero amount", onboarding.SetPriceRequest{BranchID: 1, Category: models.CategorySedan, Amount: 0}),
			Entry("negative amount", onboarding.SetPriceRequest{BranchID: 1, Category: models.CategorySedan, Amount: -5}),
			Entry("invalid category", onboarding.SetPriceRequest{BranchID: 1, Category: "BIKE", Amount: 100}),
			Entry("missing branch id", onboarding.SetPriceRequest{BranchID: 0, Category: models.CategorySedan, Amount: 100}),
		)
	})

	Describe("CreateUser", func() {
		It("registers a user", func() {
			repo.EXPECT().CreateUser(gomock.Any(), gomock.Any()).
				Return(models.User{ID: 3, Name: "Bob"}, nil)
			u, err := svc.CreateUser(ctx, onboarding.CreateUserRequest{Name: "Bob"})
			Expect(err).ToNot(HaveOccurred())
			Expect(u.ID).To(Equal(int64(3)))
		})

		It("rejects an empty name", func() {
			_, err := svc.CreateUser(ctx, onboarding.CreateUserRequest{Name: ""})
			Expect(err).To(MatchError(constants.ErrValidation))
		})
	})
})
