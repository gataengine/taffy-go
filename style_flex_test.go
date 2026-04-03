package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FlexDirection", func() {
	Describe("IsRow", func() {
		It("returns true for Row and RowReverse", func() {
			Expect(FlexDirectionRow.IsRow()).To(BeTrue())
			Expect(FlexDirectionRowReverse.IsRow()).To(BeTrue())
			Expect(FlexDirectionColumn.IsRow()).To(BeFalse())
			Expect(FlexDirectionColumnReverse.IsRow()).To(BeFalse())
		})
	})

	Describe("IsColumn", func() {
		It("returns true for Column and ColumnReverse", func() {
			Expect(FlexDirectionRow.IsColumn()).To(BeFalse())
			Expect(FlexDirectionRowReverse.IsColumn()).To(BeFalse())
			Expect(FlexDirectionColumn.IsColumn()).To(BeTrue())
			Expect(FlexDirectionColumnReverse.IsColumn()).To(BeTrue())
		})
	})

	Describe("IsReverse", func() {
		It("returns true for RowReverse and ColumnReverse", func() {
			Expect(FlexDirectionRow.IsReverse()).To(BeFalse())
			Expect(FlexDirectionRowReverse.IsReverse()).To(BeTrue())
			Expect(FlexDirectionColumn.IsReverse()).To(BeFalse())
			Expect(FlexDirectionColumnReverse.IsReverse()).To(BeTrue())
		})
	})
})
