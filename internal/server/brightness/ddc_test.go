package brightness

import (
	"testing"
)

func TestDDCBackend_PercentConversions(t *testing.T) {
	tests := []struct {
		name      string
		max       int
		percent   int
		wantValue int
	}{
		{
			name:      "1% should be 1",
			max:       100,
			percent:   1,
			wantValue: 1,
		},
		{
			name:      "50%",
			max:       100,
			percent:   50,
			wantValue: 50,
		},
		{
			name:      "100%",
			max:       100,
			percent:   100,
			wantValue: 100,
		},
		{
			name:      "0% clamped to 1",
			max:       100,
			percent:   0,
			wantValue: 1,
		},
	}

	b := &DDCBackend{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.percentToValue(tt.percent, tt.max)
			if got != tt.wantValue {
				t.Errorf("percentToValue() = %v, want %v", got, tt.wantValue)
			}
		})
	}
}

func TestDDCBackend_ValueToPercent(t *testing.T) {
	tests := []struct {
		name        string
		max         int
		value       int
		wantPercent int
		tolerance   int
	}{
		{
			name:        "min value",
			max:         100,
			value:       1,
			wantPercent: 1,
			tolerance:   0,
		},
		{
			name:        "mid value",
			max:         100,
			value:       50,
			wantPercent: 50,
			tolerance:   1,
		},
		{
			name:        "max value",
			max:         100,
			value:       100,
			wantPercent: 100,
			tolerance:   0,
		},
		{
			name:        "below min clamped",
			max:         100,
			value:       0,
			wantPercent: 1,
			tolerance:   0,
		},
	}

	b := &DDCBackend{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.valueToPercent(tt.value, tt.max)
			diff := got - tt.wantPercent
			if diff < 0 {
				diff = -diff
			}
			if diff > tt.tolerance {
				t.Errorf("valueToPercent() = %v, want %v (±%d)", got, tt.wantPercent, tt.tolerance)
			}
		})
	}
}

func TestDDCBackend_RoundTrip(t *testing.T) {
	b := &DDCBackend{}

	tests := []struct {
		name    string
		max     int
		percent int
	}{
		{"1%", 100, 1},
		{"25%", 100, 25},
		{"50%", 100, 50},
		{"75%", 100, 75},
		{"100%", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := b.percentToValue(tt.percent, tt.max)
			gotPercent := b.valueToPercent(value, tt.max)

			if diff := tt.percent - gotPercent; diff < -1 || diff > 1 {
				t.Errorf("round trip failed: wanted %d%%, got %d%% (value=%d)", tt.percent, gotPercent, value)
			}
		})
	}
}
