package timecty

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	richcty "github.com/tsarna/rich-cty-types"
	"github.com/zclconf/go-cty/cty"
)

var bg = context.Background()

// --- Timestamp.ToString ---

func TestTimestamp_ToString_WithNanos(t *testing.T) {
	ts := &Timestamp{time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)}
	s, err := ts.ToString(bg)
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:45.123456789Z", s)
}

func TestTimestamp_ToString_WholeSeconds(t *testing.T) {
	// RFC3339Nano strips trailing zeros, so a whole-second time should
	// format identically to RFC3339 (no fractional part).
	ts := &Timestamp{time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)}
	s, err := ts.ToString(bg)
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:45Z", s)
}

// --- Duration.ToString ---

func TestDuration_ToString(t *testing.T) {
	d := &Duration{90*time.Minute + 30*time.Second}
	s, err := d.ToString(bg)
	require.NoError(t, err)
	assert.Equal(t, "1h30m30s", s)
}

// --- Timestamp.Get ---

func TestTimestamp_Get(t *testing.T) {
	// 2024-01-15 (Monday) 10:30:45.123456789 UTC
	ts := &Timestamp{time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)}
	tests := []struct {
		part string
		want int64
	}{
		{"year", 2024},
		{"month", 1},
		{"day", 15},
		{"hour", 10},
		{"minute", 30},
		{"second", 45},
		{"nanosecond", 123456789},
		{"weekday", 1}, // Monday
		{"yearday", 15},
	}
	for _, tt := range tests {
		result, err := ts.Get(bg, []cty.Value{cty.StringVal(tt.part)})
		require.NoError(t, err, "get(t, %q)", tt.part)
		got, _ := result.AsBigFloat().Int64()
		assert.Equal(t, tt.want, got, "get(t, %q)", tt.part)
	}
}

func TestTimestamp_Get_ISOWeek(t *testing.T) {
	// 2024-01-01 is week 1 of ISO year 2024 (Monday)
	ts := &Timestamp{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	week, err := ts.Get(bg, []cty.Value{cty.StringVal("isoweek")})
	require.NoError(t, err)
	w, _ := week.AsBigFloat().Int64()
	assert.Equal(t, int64(1), w)

	year, err := ts.Get(bg, []cty.Value{cty.StringVal("isoyear")})
	require.NoError(t, err)
	y, _ := year.AsBigFloat().Int64()
	assert.Equal(t, int64(2024), y)
}

func TestTimestamp_Get_ISOYearBoundary(t *testing.T) {
	// 2019-12-30 is ISO week 1 of ISO year 2020
	ts := &Timestamp{time.Date(2019, 12, 30, 0, 0, 0, 0, time.UTC)}
	year, err := ts.Get(bg, []cty.Value{cty.StringVal("isoyear")})
	require.NoError(t, err)
	y, _ := year.AsBigFloat().Int64()
	assert.Equal(t, int64(2020), y)
}

func TestTimestamp_Get_UsesStoredTimezone(t *testing.T) {
	// 10:30 UTC = 05:30 New York (EST, UTC-5)
	utc := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ny, _ := time.LoadLocation("America/New_York")
	ts := &Timestamp{utc.In(ny)}
	result, err := ts.Get(bg, []cty.Value{cty.StringVal("hour")})
	require.NoError(t, err)
	got, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(5), got)
}

func TestTimestamp_Get_NoArgs_Error(t *testing.T) {
	ts := &Timestamp{time.Now()}
	_, err := ts.Get(bg, nil)
	assert.Error(t, err)
}

func TestTimestamp_Get_NonStringArg_Error(t *testing.T) {
	ts := &Timestamp{time.Now()}
	_, err := ts.Get(bg, []cty.Value{cty.NumberIntVal(1)})
	assert.Error(t, err)
}

func TestTimestamp_Get_UnknownPart_Error(t *testing.T) {
	ts := &Timestamp{time.Now()}
	_, err := ts.Get(bg, []cty.Value{cty.StringVal("quarter")})
	assert.Error(t, err)
}

// --- Duration.Get ---

func TestDuration_Get_FloatUnits(t *testing.T) {
	base := 90*time.Minute + 30*time.Second + 500*time.Millisecond
	d := &Duration{base}
	tests := []struct {
		unit string
		want float64
	}{
		{"h", base.Hours()},
		{"m", base.Minutes()},
		{"s", base.Seconds()},
	}
	for _, tt := range tests {
		result, err := d.Get(bg, []cty.Value{cty.StringVal(tt.unit)})
		require.NoError(t, err, "get(d, %q)", tt.unit)
		got, _ := result.AsBigFloat().Float64()
		assert.InDelta(t, tt.want, got, 1e-9, "get(d, %q)", tt.unit)
	}
}

func TestDuration_Get_IntUnits(t *testing.T) {
	base := 90*time.Minute + 30*time.Second + 500*time.Millisecond
	d := &Duration{base}
	tests := []struct {
		unit string
		want int64
	}{
		{"ms", base.Milliseconds()},
		{"us", base.Microseconds()},
		{"ns", base.Nanoseconds()},
	}
	for _, tt := range tests {
		result, err := d.Get(bg, []cty.Value{cty.StringVal(tt.unit)})
		require.NoError(t, err, "get(d, %q)", tt.unit)
		got, _ := result.AsBigFloat().Int64()
		assert.Equal(t, tt.want, got, "get(d, %q)", tt.unit)
	}
}

func TestDuration_Get_NoArgs_Error(t *testing.T) {
	d := &Duration{time.Minute}
	_, err := d.Get(bg, nil)
	assert.Error(t, err)
}

func TestDuration_Get_NonStringArg_Error(t *testing.T) {
	d := &Duration{time.Minute}
	_, err := d.Get(bg, []cty.Value{cty.NumberIntVal(1)})
	assert.Error(t, err)
}

func TestDuration_Get_UnknownUnit_Error(t *testing.T) {
	d := &Duration{time.Minute}
	_, err := d.Get(bg, []cty.Value{cty.StringVal("d")})
	assert.Error(t, err)
}

// --- Integration: dispatch through rich-cty-types generic functions ---

func TestGenericTostring_Time(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC))
	fn := richcty.GetGenericFunctions()["tostring"]
	result, err := fn.Call([]cty.Value{ts})
	require.NoError(t, err)
	assert.Equal(t, cty.StringVal("2024-01-15T10:30:45Z"), result)
}

func TestGenericTostring_Duration(t *testing.T) {
	d := NewDurationCapsule(90*time.Minute + 30*time.Second)
	fn := richcty.GetGenericFunctions()["tostring"]
	result, err := fn.Call([]cty.Value{d})
	require.NoError(t, err)
	assert.Equal(t, cty.StringVal("1h30m30s"), result)
}

func TestGenericGet_Time(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC))
	fn := richcty.GetGenericFunctions()["get"]
	result, err := fn.Call([]cty.Value{ts, cty.StringVal("year")})
	require.NoError(t, err)
	got, _ := result.AsBigFloat().Int64()
	assert.Equal(t, int64(2024), got)
}

func TestGenericGet_Duration(t *testing.T) {
	d := NewDurationCapsule(2 * time.Hour)
	fn := richcty.GetGenericFunctions()["get"]
	result, err := fn.Call([]cty.Value{d, cty.StringVal("s")})
	require.NoError(t, err)
	got, _ := result.AsBigFloat().Float64()
	assert.InDelta(t, 7200.0, got, 1e-9)
}
