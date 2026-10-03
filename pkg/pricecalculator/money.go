package pricecalculator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MoneyScale is the number of minor units per major currency unit.
// Prices and price_step are stored as integer minor units (2 decimal places).
const MoneyScale int64 = 100

// MinPriceStepMajor is the smallest allowed non-zero price_step in major units.
const MinPriceStepMajor = 0.1

// Money is a currency amount in minor units (1/100 of a major unit).
type Money int64

// Maj converts a whole major-unit amount to Money (tests / helpers).
func Maj(major int64) Money {
	return Money(major * MoneyScale)
}

// Majf converts a major-unit float to Money.
func Majf(major float64) Money {
	return Money(math.Round(major * float64(MoneyScale)))
}

// Major returns the amount in major currency units.
func (m Money) Major() float64 {
	return float64(m) / float64(MoneyScale)
}

func (m Money) String() string {
	major := int64(m) / MoneyScale
	minor := int64(m) % MoneyScale
	if minor < 0 {
		minor = -minor
	}
	if minor == 0 {
		return strconv.FormatInt(major, 10)
	}
	return fmt.Sprintf("%d.%02d", major, minor)
}

func (m Money) MarshalJSON() ([]byte, error) {
	if m%Money(MoneyScale) == 0 {
		return json.Marshal(int64(m) / MoneyScale)
	}
	// Emit a compact decimal (trim trailing zeros via FormatFloat 'f' precision).
	return []byte(strconv.FormatFloat(m.Major(), 'f', -1, 64)), nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*m = 0
		return nil
	}

	// Prefer decimal string parsing to avoid float binary artifacts (e.g. 0.1).
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("money: %w", err)
	}

	parsed, err := parseMajorToMoney(num.String())
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

func parseMajorToMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("money: empty value")
	}

	negative := false
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	} else if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	parts := strings.SplitN(s, ".", 2)
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	major, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money: invalid value %q", s)
	}

	var frac int64
	if len(parts) == 2 {
		fracStr := parts[1]
		if fracStr == "" {
			return 0, fmt.Errorf("money: invalid value %q", s)
		}
		if len(fracStr) > 2 {
			// Round half-up from the third decimal digit.
			extra := fracStr[2]
			fracStr = fracStr[:2]
			frac, err = strconv.ParseInt(fracStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("money: invalid value %q", s)
			}
			if extra >= '5' {
				frac++
			}
		} else {
			for len(fracStr) < 2 {
				fracStr += "0"
			}
			frac, err = strconv.ParseInt(fracStr, 10, 64)
			if err != nil {
				return 0, fmt.Errorf("money: invalid value %q", s)
			}
		}
	}

	if frac >= MoneyScale {
		major++
		frac -= MoneyScale
	}

	minor := major*MoneyScale + frac
	if negative {
		minor = -minor
	}
	return Money(minor), nil
}
