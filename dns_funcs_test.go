package timecty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- nextzoneserial ---

func TestNextZoneSerialNewDay(t *testing.T) {
	// Serial from a previous day: should return first serial of today.
	ts := NewTimeCapsule(time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026012200), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026012300), n)
}

func TestNextZoneSerialSameDay(t *testing.T) {
	// Serial already within today: should increment.
	ts := NewTimeCapsule(time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026012305), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026012306), n)
}

func TestNextZoneSerialRollover(t *testing.T) {
	// 100th update on a day causes NN to overflow into next day's range; still valid.
	ts := NewTimeCapsule(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026123199), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026123200), n)
}

func TestNextZoneSerialStringSerial(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.StringVal("2026012205"), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026012300), n)
}

func TestNextZoneSerialZeroSerial(t *testing.T) {
	// nextzoneserial(0, t) returns the first serial of the day.
	ts := NewTimeCapsule(time.Date(2026, 1, 23, 0, 0, 0, 0, time.UTC))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(0), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026012300), n)
}

func TestNextZoneSerialUsesTimezone(t *testing.T) {
	// 23:30 UTC on Jan 22 = 18:30 New York (EST, UTC-5) → still Jan 22 in NY.
	ny, _ := time.LoadLocation("America/New_York")
	ts := NewTimeCapsule(time.Date(2026, 1, 22, 23, 30, 0, 0, time.UTC).In(ny))
	result, err := NextZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(0), ts})
	require.NoError(t, err)
	n, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2026012200), n) // NY date is Jan 22, not Jan 23
}

// --- parsezoneserial ---

func TestParseZoneSerialNormal(t *testing.T) {
	result, err := ParseZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026012307)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 23, got.Day())
}

func TestParseZoneSerialString(t *testing.T) {
	result, err := ParseZoneSerialFunc.Call([]cty.Value{cty.StringVal("2026012300")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 23, got.Day())
}

func TestParseZoneSerialInvalidMonth(t *testing.T) {
	// Month 13 → December 31.
	result, err := ParseZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026133200)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.December, got.Month())
	assert.Equal(t, 31, got.Day())
}

func TestParseZoneSerialInvalidDay(t *testing.T) {
	// February 30 → February 28 (2026 is not a leap year).
	result, err := ParseZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026023000)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.February, got.Month())
	assert.Equal(t, 28, got.Day())
}

func TestParseZoneSerialRollover(t *testing.T) {
	// 2026-12-31 + 100 updates → serial 2026123200, which looks like day 32 of Dec.
	// Dec has 31 days, so snap to Dec 31.
	result, err := ParseZoneSerialFunc.Call([]cty.Value{cty.NumberIntVal(2026123200)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.December, got.Month())
	assert.Equal(t, 31, got.Day())
}
