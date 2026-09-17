package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestScriptCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expense Sharing CLI Suite")
}
