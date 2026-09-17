package expensesharing_test

import (
	"strconv"

	"go-starter/internal/logger"
)

// newTestLogger returns a real logger at error level. The service logs on the
// repository-error path, so a nil logger would panic there; this keeps the
// specs exercising the real code path without extra mocking.
func newTestLogger() logger.Logger {
	return logger.New("error")
}

// nameFor gives a deterministic display name for a user id, e.g. 1 -> "User1".
func nameFor(id int64) string {
	return "User" + strconv.FormatInt(id, 10)
}
