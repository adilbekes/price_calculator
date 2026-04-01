package main

import (
	"fmt"
	"log"
	"price_calculator/pkg/pricecalculator"
	"strings"
)

func formatPeriods(periods []pricecalculator.PricingPeriod) string {
	parts := make([]string, len(periods))
	for i, period := range periods {
		parts[i] = period.String()
	}
	return strings.Join(parts, ", ")
}

func formatBreakdown(items []pricecalculator.BreakdownItem) string {
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.String()
	}
	return strings.Join(parts, ", ")
}

func main() {
	calc := pricecalculator.NewCalculator()
	baseRequest := pricecalculator.CalculateRequest{
		RequestedDurationMinutes: 130,
		Periods: []pricecalculator.PricingPeriod{
			{DurationMinutes: 60, Price: 1000},
			//{DurationMinutes: 120, Price: 1800},
			//{DurationMinutes: 180, Price: 2500},
		},
	}

	modes := []pricecalculator.PricingMode{
		pricecalculator.PricingModeRoundUp,
		pricecalculator.PricingModeProrateMinimum,
		pricecalculator.PricingModeProrateAny,
		pricecalculator.PricingModeRoundUpMinimumAndProrateAny,
	}

	fmt.Printf("Requested duration: %d minutes\n", baseRequest.RequestedDurationMinutes)
	fmt.Printf("Periods: %s\n\n", formatPeriods(baseRequest.Periods))

	for _, mode := range modes {
		request := baseRequest
		request.PricingMode = mode

		result, err := calc.Calculate(request)
		if err != nil {
			log.Fatalf("%s: %v", mode.String(), err)
		}

		fmt.Printf("Mode: %s \n", mode.String())
		fmt.Printf("  Total price: %d💰\n", result.TotalPrice)
		fmt.Printf("  Covered minutes: %d⏱\n", result.CoveredMinutes)
		fmt.Printf("  Breakdown: %s\n\n", formatBreakdown(result.Breakdown))
	}
}
