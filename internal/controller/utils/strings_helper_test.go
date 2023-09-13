package controllers_utils

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("StringsHelper", func() {

	Describe("ContainsString", func() {

		It("should return true when string is present in the slice", func() {
			slice := []string{"apple", "banana", "cherry"}
			Expect(ContainsString(slice, "banana")).To(BeTrue())
		})

		It("should return false when string is not present in the slice", func() {
			slice := []string{"apple", "banana", "cherry"}
			Expect(ContainsString(slice, "grape")).To(BeFalse())
		})

		It("should return false when slice is empty", func() {
			slice := []string{}
			Expect(ContainsString(slice, "apple")).To(BeFalse())
		})

	})

	Describe("RemoveString", func() {

		It("should remove the given string from the slice", func() {
			slice := []string{"apple", "banana", "cherry"}
			expected := []string{"apple", "cherry"}
			Expect(RemoveString(slice, "banana")).To(Equal(expected))
		})

		It("should return the same slice if string is not present", func() {
			slice := []string{"apple", "banana", "cherry"}
			expected := []string{"apple", "banana", "cherry"}
			Expect(RemoveString(slice, "grape")).To(Equal(expected))
		})

		It("should return an empty slice if input slice is empty", func() {
			slice := []string{}
			expected := []string{}
			Expect(RemoveString(slice, "apple")).To(Equal(expected))
		})

	})

})
