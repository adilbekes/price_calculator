package pricecalculator

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoney_JSONRoundTrip_IntAndFloat(t *testing.T) {
	cases := []struct {
		raw   string
		major float64
	}{
		{`1000`, 1000},
		{`1000.1`, 1000.1},
		{`0.1`, 0.1},
		{`1178.5`, 1178.5},
	}
	for _, tc := range cases {
		var m Money
		require.NoError(t, json.Unmarshal([]byte(tc.raw), &m))
		assert.InDelta(t, tc.major, m.Major(), 1e-9, tc.raw)

		encoded, err := json.Marshal(m)
		require.NoError(t, err)
		var again Money
		require.NoError(t, json.Unmarshal(encoded, &again))
		assert.Equal(t, m, again, tc.raw)
	}
}

func TestMoney_Unmarshal_RejectsInvalid(t *testing.T) {
	var m Money
	assert.Error(t, json.Unmarshal([]byte(`"x"`), &m))
}
