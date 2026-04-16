package timecty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- duration ---

func TestDurationGoFormat(t *testing.T) {
	result, err := DurationFunc.Call([]cty.Value{cty.StringVal("5m30s")})
	require.NoError(t, err)
	assert.Equal(t, DurationCapsuleType, result.Type())
	d, _ := GetDuration(result)
	assert.Equal(t, 5*time.Minute+30*time.Second, d)
}

func TestDurationISO8601(t *testing.T) {
	result, err := DurationFunc.Call([]cty.Value{cty.StringVal("PT5M")})
	require.NoError(t, err)
	d, _ := GetDuration(result)
	assert.Equal(t, 5*time.Minute, d)
}

func TestDurationISO8601Complex(t *testing.T) {
	result, err := DurationFunc.Call([]cty.Value{cty.StringVal("PT1H30M")})
	require.NoError(t, err)
	d, _ := GetDuration(result)
	assert.Equal(t, 90*time.Minute, d)
}

func TestDurationCalendarError(t *testing.T) {
	_, err := DurationFunc.Call([]cty.Value{cty.StringVal("P1Y")})
	assert.Error(t, err)

	_, err = DurationFunc.Call([]cty.Value{cty.StringVal("P1M")})
	assert.Error(t, err)
}

func TestDurationFromNumber(t *testing.T) {
	tests := []struct {
		n    float64
		unit string
		want time.Duration
	}{
		{5, "h", 5 * time.Hour},
		{30, "m", 30 * time.Minute},
		{10, "s", 10 * time.Second},
		{500, "ms", 500 * time.Millisecond},
		{1000, "us", 1000 * time.Microsecond},
		{1000000, "ns", 1000000 * time.Nanosecond},
		{1.5, "s", 1500 * time.Millisecond},
	}
	for _, tt := range tests {
		result, err := DurationFunc.Call([]cty.Value{
			cty.NumberFloatVal(tt.n),
			cty.StringVal(tt.unit),
		})
		require.NoError(t, err, "duration(%v, %q)", tt.n, tt.unit)
		d, _ := GetDuration(result)
		assert.Equal(t, tt.want, d, "duration(%v, %q)", tt.n, tt.unit)
	}
}

func TestDurationInvalidUnit(t *testing.T) {
	_, err := DurationFunc.Call([]cty.Value{cty.NumberIntVal(5), cty.StringVal("days")})
	assert.Error(t, err)
}

// --- formatduration ---

func TestFormatDurationGo(t *testing.T) {
	dur := NewDurationCapsule(90 * time.Minute)
	result, err := FormatDurationFunc.Call([]cty.Value{dur})
	require.NoError(t, err)
	assert.Equal(t, "1h30m0s", result.AsString())
}

func TestFormatDurationGoExplicit(t *testing.T) {
	dur := NewDurationCapsule(90 * time.Minute)
	result, err := FormatDurationFunc.Call([]cty.Value{dur, cty.StringVal("go")})
	require.NoError(t, err)
	assert.Equal(t, "1h30m0s", result.AsString())
}

func TestFormatDurationISO(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "PT0S"},
		{5 * time.Minute, "PT5M"},
		{90 * time.Minute, "PT1H30M"},
		{time.Hour + 30*time.Minute + 15*time.Second, "PT1H30M15S"},
		{500 * time.Millisecond, "PT0.5S"},
		{-5 * time.Minute, "-PT5M"},
	}
	for _, tt := range tests {
		dur := NewDurationCapsule(tt.d)
		result, err := FormatDurationFunc.Call([]cty.Value{dur, cty.StringVal("iso")})
		require.NoError(t, err, "formatduration(%v, \"iso\")", tt.d)
		assert.Equal(t, tt.want, result.AsString(), "formatduration(%v, \"iso\")", tt.d)
	}
}

func TestFormatDurationInvalidFormat(t *testing.T) {
	dur := NewDurationCapsule(time.Minute)
	_, err := FormatDurationFunc.Call([]cty.Value{dur, cty.StringVal("invalid")})
	assert.Error(t, err)
}

// --- absduration ---

func TestAbsDurationPositive(t *testing.T) {
	d := NewDurationCapsule(5 * time.Minute)
	result, err := AbsDurationFunc.Call([]cty.Value{d})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 5*time.Minute, got)
}

func TestAbsDurationNegative(t *testing.T) {
	d := NewDurationCapsule(-5 * time.Minute)
	result, err := AbsDurationFunc.Call([]cty.Value{d})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 5*time.Minute, got)
}

// --- duration arithmetic ---

func TestDurationAdd(t *testing.T) {
	d1 := NewDurationCapsule(30 * time.Minute)
	d2 := NewDurationCapsule(45 * time.Minute)
	result, err := DurationAddFunc.Call([]cty.Value{d1, d2})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 75*time.Minute, got)
}

func TestDurationSub(t *testing.T) {
	d1 := NewDurationCapsule(2 * time.Hour)
	d2 := NewDurationCapsule(30 * time.Minute)
	result, err := DurationSubFunc.Call([]cty.Value{d1, d2})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 90*time.Minute, got)
}

func TestDurationMul(t *testing.T) {
	d := NewDurationCapsule(30 * time.Minute)
	result, err := DurationMulFunc.Call([]cty.Value{d, cty.NumberIntVal(3)})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 90*time.Minute, got)
}

func TestDurationMulFractional(t *testing.T) {
	d := NewDurationCapsule(time.Hour)
	result, err := DurationMulFunc.Call([]cty.Value{d, cty.NumberFloatVal(1.5)})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 90*time.Minute, got)
}

func TestDurationDiv(t *testing.T) {
	d := NewDurationCapsule(time.Hour)
	result, err := DurationDivFunc.Call([]cty.Value{d, cty.NumberIntVal(4)})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 15*time.Minute, got)
}

func TestDurationDivByZero(t *testing.T) {
	d := NewDurationCapsule(time.Hour)
	_, err := DurationDivFunc.Call([]cty.Value{d, cty.NumberIntVal(0)})
	assert.Error(t, err)
}

func TestDurationTruncate(t *testing.T) {
	d := NewDurationCapsule(1*time.Hour + 37*time.Minute + 42*time.Second)
	m := NewDurationCapsule(time.Minute)
	result, err := DurationTruncateFunc.Call([]cty.Value{d, m})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 1*time.Hour+37*time.Minute, got)
}

func TestDurationRound(t *testing.T) {
	d := NewDurationCapsule(1*time.Hour + 37*time.Minute + 42*time.Second)
	m := NewDurationCapsule(time.Minute)
	result, err := DurationRoundFunc.Call([]cty.Value{d, m})
	require.NoError(t, err)
	got, _ := GetDuration(result)
	assert.Equal(t, 1*time.Hour+38*time.Minute, got)
}

// --- duration comparison ---

func TestDurationLt(t *testing.T) {
	d1 := NewDurationCapsule(5 * time.Minute)
	d2 := NewDurationCapsule(10 * time.Minute)

	result, err := DurationLtFunc.Call([]cty.Value{d1, d2})
	require.NoError(t, err)
	assert.True(t, result.True())

	result, err = DurationLtFunc.Call([]cty.Value{d2, d1})
	require.NoError(t, err)
	assert.False(t, result.True())

	// Equal: not less than
	result, err = DurationLtFunc.Call([]cty.Value{d1, d1})
	require.NoError(t, err)
	assert.False(t, result.True())
}

func TestDurationGt(t *testing.T) {
	d1 := NewDurationCapsule(5 * time.Minute)
	d2 := NewDurationCapsule(10 * time.Minute)

	result, err := DurationGtFunc.Call([]cty.Value{d2, d1})
	require.NoError(t, err)
	assert.True(t, result.True())

	result, err = DurationGtFunc.Call([]cty.Value{d1, d2})
	require.NoError(t, err)
	assert.False(t, result.True())
}

// --- capsule equality ---

func TestDurationCapsuleEquality(t *testing.T) {
	d1 := NewDurationCapsule(5 * time.Minute)
	d2 := NewDurationCapsule(10 * time.Minute)
	d3 := NewDurationCapsule(5 * time.Minute)

	assert.True(t, d1.Equals(d3).True())
	assert.True(t, d1.Equals(d2).False())
}

// --- durationToISO8601 helper ---

func TestDurationToISO8601(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "PT0S"},
		{time.Second, "PT1S"},
		{time.Minute, "PT1M"},
		{time.Hour, "PT1H"},
		{24 * time.Hour, "PT24H"},
		{time.Hour + 30*time.Minute + 45*time.Second, "PT1H30M45S"},
		{500 * time.Millisecond, "PT0.5S"},
		{1500 * time.Millisecond, "PT1.5S"},
		{time.Microsecond, "PT0.000001S"},
		{time.Nanosecond, "PT0.000000001S"},
		{-time.Minute, "-PT1M"},
	}
	for _, tt := range tests {
		got := durationToISO8601(tt.d)
		assert.Equal(t, tt.want, got, "durationToISO8601(%v)", tt.d)
	}
}
