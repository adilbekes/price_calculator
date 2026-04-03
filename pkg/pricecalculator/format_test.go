package pricecalculator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStringer string

func (s testStringer) String() string {
	return string(s)
}

func TestFormatItems_EmptySlice_ReturnsEmptyString(t *testing.T) {
	assert.Equal(t, "", FormatItems([]testStringer{}))
}

func TestFormatItems_SingleItem_ReturnsSingleString(t *testing.T) {
	assert.Equal(t, "one", FormatItems([]testStringer{"one"}))
}

func TestFormatItems_MultipleItems_JoinsWithCommaSpace(t *testing.T) {
	assert.Equal(t, "one, two, three", FormatItems([]testStringer{"one", "two", "three"}))
}

func TestFormatItems_WithPricingPeriods_UsesStringRepresentation(t *testing.T) {
	items := []PricingPeriod{{DurationMinutes: 60, Price: 1000}, {DurationMinutes: 120, Price: 1800}}
	assert.Equal(t, "60⏱ - 1000💰, 120⏱ - 1800💰", FormatItems(items))
}

func TestFormatItems_WithBreakdownItems_UsesStringRepresentation(t *testing.T) {
	items := []BreakdownItem{{Quantity: 1, DurationMinutes: 60, UsedDuration: 60, Price: 1000, UsedPrice: 1000}}
	assert.Equal(t, "1x[60⏱ - 1000💰]", FormatItems(items))
}

