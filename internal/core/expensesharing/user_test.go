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

var _ = Describe("AddUser", func() {
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

	validReq := expensesharing.AddUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
		Phone: "9876543210",
	}

	Describe("happy path", func() {
		It("creates the user and returns it with the DB-assigned id", func() {
			repo.EXPECT().
				CreateUser(gomock.Any(), models.User{
					Name:  "Alice",
					Email: "alice@example.com",
					Phone: "9876543210",
				}).
				Return(models.User{
					ID:    1,
					Name:  "Alice",
					Email: "alice@example.com",
					Phone: "9876543210",
				}, nil)

			user, err := svc.AddUser(ctx, validReq)

			Expect(err).ToNot(HaveOccurred())
			Expect(user.ID).To(Equal(int64(1)))
			Expect(user.Name).To(Equal("Alice"))
			Expect(user.Email).To(Equal("alice@example.com"))
			Expect(user.Phone).To(Equal("9876543210"))
		})
	})

	Describe("edge cases", func() {
		It("trims surrounding whitespace before persisting", func() {
			repo.EXPECT().
				CreateUser(gomock.Any(), models.User{
					Name:  "Alice",
					Email: "alice@example.com",
					Phone: "9876543210",
				}).
				Return(models.User{ID: 1, Name: "Alice", Email: "alice@example.com", Phone: "9876543210"}, nil)

			_, err := svc.AddUser(ctx, expensesharing.AddUserRequest{
				Name:  "  Alice  ",
				Email: "  alice@example.com  ",
				Phone: "  9876543210  ",
			})

			Expect(err).ToNot(HaveOccurred())
		})

		It("treats a whitespace-only name as empty", func() {
			_, err := svc.AddUser(ctx, expensesharing.AddUserRequest{
				Name:  "   ",
				Email: "alice@example.com",
				Phone: "9876543210",
			})
			Expect(err).To(MatchError(constants.ErrEmptyName))
		})
	})

	Describe("invalid input", func() {
		DescribeTable("returns the matching validation sentinel",
			func(req expensesharing.AddUserRequest, expected error) {
				// No repository call is expected: validation fails first.
				_, err := svc.AddUser(ctx, req)
				Expect(err).To(MatchError(expected))
			},
			Entry("empty name",
				expensesharing.AddUserRequest{Name: "", Email: "a@b.com", Phone: "9876543210"},
				constants.ErrEmptyName),
			Entry("empty email",
				expensesharing.AddUserRequest{Name: "Alice", Email: "", Phone: "9876543210"},
				constants.ErrEmptyEmail),
			Entry("empty phone",
				expensesharing.AddUserRequest{Name: "Alice", Email: "a@b.com", Phone: ""},
				constants.ErrEmptyPhone),
			Entry("email without @",
				expensesharing.AddUserRequest{Name: "Alice", Email: "alice.example.com", Phone: "9876543210"},
				constants.ErrInvalidEmail),
			Entry("email without domain dot",
				expensesharing.AddUserRequest{Name: "Alice", Email: "alice@example", Phone: "9876543210"},
				constants.ErrInvalidEmail),
			Entry("phone with letters",
				expensesharing.AddUserRequest{Name: "Alice", Email: "a@b.com", Phone: "98765abcde"},
				constants.ErrInvalidPhone),
			Entry("phone too short",
				expensesharing.AddUserRequest{Name: "Alice", Email: "a@b.com", Phone: "12345"},
				constants.ErrInvalidPhone),
			Entry("phone too long",
				expensesharing.AddUserRequest{Name: "Alice", Email: "a@b.com", Phone: "12345678901"},
				constants.ErrInvalidPhone),
		)

		It("checks emptiness before format (blank email reports required, not invalid)", func() {
			_, err := svc.AddUser(ctx, expensesharing.AddUserRequest{
				Name:  "Alice",
				Email: "   ",
				Phone: "9876543210",
			})
			Expect(err).To(MatchError(constants.ErrEmptyEmail))
		})
	})

	Describe("duplicate users", func() {
		It("returns DuplicateEmail when the repository reports a taken email", func() {
			repo.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(models.User{}, constants.DuplicateEmail)

			_, err := svc.AddUser(ctx, validReq)
			Expect(err).To(MatchError(constants.DuplicateEmail))
		})

		It("returns DuplicatePhone when the repository reports a taken phone", func() {
			repo.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(models.User{}, constants.DuplicatePhone)

			_, err := svc.AddUser(ctx, validReq)
			Expect(err).To(MatchError(constants.DuplicatePhone))
		})

		It("propagates an unexpected repository error unchanged", func() {
			boom := errors.New("connection reset")
			repo.EXPECT().
				CreateUser(gomock.Any(), gomock.Any()).
				Return(models.User{}, boom)

			_, err := svc.AddUser(ctx, validReq)
			Expect(err).To(MatchError(boom))
		})
	})
})
