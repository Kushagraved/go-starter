package expensesharing

import (
	"context"
	"regexp"
	"strings"

	"go-starter/internal/constants"
	"go-starter/internal/logger"
	"go-starter/internal/models"
)

// AddUserRequest is the validated input for creating a user. It mirrors the
// Add_User CLI command: Add_User [Name] [Email] [PhoneNumber].
type AddUserRequest struct {
	Name  string
	Email string
	Phone string
}

// emailRegex is a pragmatic address check: non-space local part, an @, a domain
// with at least one dot. It rejects the obvious garbage without pretending to
// implement the full RFC 5322 grammar.
var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// phoneRegex accepts exactly 10 digits, the common local mobile format.
var phoneRegex = regexp.MustCompile(`^[0-9]{10}$`)

// AddUser validates the request and persists a new user.
//
// The DB assigns the id (bigserial), so none is generated here. Uniqueness of
// email and phone is enforced by the database; the repository maps a violation
// to constants.DuplicateEmail / DuplicatePhone, which this method passes through.
func (s *service) AddUser(ctx context.Context, req AddUserRequest) (models.User, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	if err := validateAddUser(req); err != nil {
		return models.User{}, err
	}

	user, err := s.repo.CreateUser(ctx, models.User{
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	})
	if err != nil {
		s.logger.Error(ctx, "ExpenseSharing", "AddUser", "failed to create user",
			logger.NewString("email", req.Email), logger.NewError(err))
		return models.User{}, err
	}
	return user, nil
}

// validateAddUser checks required fields and format. Order is deliberate:
// emptiness before format, so a blank field reports "required" rather than
// "invalid".
func validateAddUser(req AddUserRequest) error {
	switch {
	case req.Name == "":
		return constants.ErrEmptyName
	case req.Email == "":
		return constants.ErrEmptyEmail
	case req.Phone == "":
		return constants.ErrEmptyPhone
	case !emailRegex.MatchString(req.Email):
		return constants.ErrInvalidEmail
	case !phoneRegex.MatchString(req.Phone):
		return constants.ErrInvalidPhone
	default:
		return nil
	}
}
