package onboarding_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestOnboarding(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Onboarding Core Suite")
}
