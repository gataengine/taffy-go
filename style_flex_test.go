package taffy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/gataengine/taffy-go/style"
)

var _ = Describe("FlexDirection", func() {
	Describe("IsRow", func() {
		It("returns true for Row and RowReverse", func() {
			Expect(style.FlexDirectionRow.IsRow()).To(BeTrue())
			Expect(style.FlexDirectionRowReverse.IsRow()).To(BeTrue())
			Expect(style.FlexDirectionColumn.IsRow()).To(BeFalse())
			Expect(style.FlexDirectionColumnReverse.IsRow()).To(BeFalse())
		})
	})

	Describe("IsColumn", func() {
		It("returns true for Column and ColumnReverse", func() {
			Expect(style.FlexDirectionRow.IsColumn()).To(BeFalse())
			Expect(style.FlexDirectionRowReverse.IsColumn()).To(BeFalse())
			Expect(style.FlexDirectionColumn.IsColumn()).To(BeTrue())
			Expect(style.FlexDirectionColumnReverse.IsColumn()).To(BeTrue())
		})
	})

	Describe("IsReverse", func() {
		It("returns true for RowReverse and ColumnReverse", func() {
			Expect(style.FlexDirectionRow.IsReverse()).To(BeFalse())
			Expect(style.FlexDirectionRowReverse.IsReverse()).To(BeTrue())
			Expect(style.FlexDirectionColumn.IsReverse()).To(BeFalse())
			Expect(style.FlexDirectionColumnReverse.IsReverse()).To(BeTrue())
		})
	})
})
