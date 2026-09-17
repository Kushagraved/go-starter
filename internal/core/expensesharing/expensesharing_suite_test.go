package expensesharing_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExpenseSharing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expense Sharing Core Suite")
}
