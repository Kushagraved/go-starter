package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("money conversions", func() {
	Describe("rupeesToPaisa", func() {
		DescribeTable("parses valid rupee strings",
			func(in string, want int64) {
				got, err := rupeesToPaisa(in)
				Expect(err).ToNot(HaveOccurred())
				Expect(got).To(Equal(want))
			},
			Entry("whole rupees", "100", int64(10000)),
			Entry("two decimals", "33.34", int64(3334)),
			Entry("one decimal pads", "0.5", int64(50)),
			Entry("zero", "0", int64(0)),
			Entry("exact paisa", "0.01", int64(1)),
			Entry("large", "12345.67", int64(1234567)),
		)

		DescribeTable("rejects invalid input",
			func(in string) {
				_, err := rupeesToPaisa(in)
				Expect(err).To(HaveOccurred())
			},
			Entry("empty", ""),
			Entry("negative", "-5"),
			Entry("three decimals", "1.234"),
			Entry("letters", "abc"),
			Entry("trailing dot", "5."),
		)
	})

	Describe("paisaToRupees", func() {
		DescribeTable("formats paisa as two-decimal rupees",
			func(in int64, want string) {
				Expect(paisaToRupees(in)).To(Equal(want))
			},
			Entry("round", int64(10000), "100.00"),
			Entry("remainder", int64(3334), "33.34"),
			Entry("sub-rupee", int64(50), "0.50"),
			Entry("single paisa", int64(1), "0.01"),
			Entry("negative", int64(-3333), "-33.33"),
		)
	})

	Describe("percentToBasisPoints", func() {
		It("scales percent like rupees to paisa", func() {
			bp, err := percentToBasisPoints("33.33")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp).To(Equal(int64(3333)))

			bp, err = percentToBasisPoints("100")
			Expect(err).ToNot(HaveOccurred())
			Expect(bp).To(Equal(int64(10000)))
		})
	})
})
