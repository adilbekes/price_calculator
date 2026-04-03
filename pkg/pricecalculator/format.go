package pricecalculator

import (
	"fmt"
	"strings"
)

func FormatItems[T fmt.Stringer](items []T) string {
	if len(items) == 0 {
		return ""
	}

	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.String()
	}

	return strings.Join(parts, ", ")
}
