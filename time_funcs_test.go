package timecty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// --- now ---

func TestNowNoArgs(t *testing.T) {
	before := time.Now()
	result, err := NowFunc.Call([]cty.Value{})
	after := time.Now()
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.True(t, !got.Before(before) && !got.After(after))
}

func TestNowUTC(t *testing.T) {
	result, err := NowFunc.Call([]cty.Value{cty.StringVal("UTC")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, "UTC", got.Location().String())
}

func TestNowNamedTZ(t *testing.T) {
	result, err := NowFunc.Call([]cty.Value{cty.StringVal("America/New_York")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, "America/New_York", got.Location().String())
}

func TestNowInvalidTZ(t *testing.T) {
	_, err := NowFunc.Call([]cty.Value{cty.StringVal("Not/ATimezone")})
	assert.Error(t, err)
}

// --- parsetime ---

func TestParseTimeRFC3339(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{cty.StringVal("2024-01-15T10:30:00Z")})
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 15, got.Day())
	assert.Equal(t, 10, got.Hour())
	assert.Equal(t, 30, got.Minute())
	assert.Equal(t, 0, got.Second())
	assert.Equal(t, "UTC", got.Location().String())
}

func TestParseTimeRFC3339Nano(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{cty.StringVal("2024-01-15T10:30:00.123456789Z")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 123456789, got.Nanosecond())
}

func TestParseTimeWithOffset(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{cty.StringVal("2024-01-15T10:30:00+05:30")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	_, offset := got.Zone()
	assert.Equal(t, 5*3600+30*60, offset)
}

func TestParseTimeInvalid(t *testing.T) {
	_, err := ParseTimeFunc.Call([]cty.Value{cty.StringVal("not a time")})
	assert.Error(t, err)
}

func TestParseTimeOneArg(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{cty.StringVal("2024-01-15T10:30:00Z")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), got)
}

func TestParseTimeTwoArgs(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{
		cty.StringVal("2006-01-02"),
		cty.StringVal("2024-01-15"),
	})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC), got)
}

func TestParseTimeTwoArgsNamedFormat(t *testing.T) {
	result, err := ParseTimeFunc.Call([]cty.Value{
		cty.StringVal("@rfc3339"),
		cty.StringVal("2024-01-15T10:30:00Z"),
	})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), got)
}

func TestParseTimeThreeArgs(t *testing.T) {
	// parse date-only string with explicit timezone
	result, err := ParseTimeFunc.Call([]cty.Value{
		cty.StringVal("2006-01-02"),
		cty.StringVal("2024-01-15"),
		cty.StringVal("America/New_York"),
	})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, "America/New_York", got.Location().String())
	assert.Equal(t, 15, got.Day())
}

func TestParseTimeInvalidFormat(t *testing.T) {
	_, err := ParseTimeFunc.Call([]cty.Value{
		cty.StringVal("@bogus"),
		cty.StringVal("2024-01-15"),
	})
	assert.Error(t, err)
}

// --- timeadd ---

func TestTimeAddStringString(t *testing.T) {
	// Backward-compatible string/string form
	result, err := TimeAddFunc.Call([]cty.Value{
		cty.StringVal("2024-01-15T10:30:00Z"),
		cty.StringVal("1h"),
	})
	require.NoError(t, err)
	assert.Equal(t, cty.String, result.Type())
	assert.Equal(t, "2024-01-15T11:30:00Z", result.AsString())
}

func TestTimeAddTimeDuration(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	dur := NewDurationCapsule(time.Hour)
	result, err := TimeAddFunc.Call([]cty.Value{ts, dur})
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.Equal(t, 11, got.Hour())
}

func TestTimeAddTimeString(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := TimeAddFunc.Call([]cty.Value{ts, cty.StringVal("30m")})
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.Equal(t, 11, got.Hour())
	assert.Equal(t, 0, got.Minute())
}

func TestTimeAddStringDuration(t *testing.T) {
	dur := NewDurationCapsule(time.Hour)
	result, err := TimeAddFunc.Call([]cty.Value{
		cty.StringVal("2024-01-15T10:30:00Z"),
		dur,
	})
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.Equal(t, 11, got.Hour())
}

// --- timesub ---

func TestTimeSubTimesReturnsDuration(t *testing.T) {
	t1 := NewTimeCapsule(time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC))
	t2 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := TimeSubFunc.Call([]cty.Value{t1, t2})
	require.NoError(t, err)
	assert.Equal(t, DurationCapsuleType, result.Type())
	d, _ := GetDuration(result)
	assert.Equal(t, time.Hour, d)
}

func TestTimeSubTimesNegative(t *testing.T) {
	t1 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	t2 := NewTimeCapsule(time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC))
	result, err := TimeSubFunc.Call([]cty.Value{t1, t2})
	require.NoError(t, err)
	d, _ := GetDuration(result)
	assert.Equal(t, -time.Hour, d)
}

func TestTimeSubTimeDurationReturnsTime(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC))
	dur := NewDurationCapsule(time.Hour)
	result, err := TimeSubFunc.Call([]cty.Value{ts, dur})
	require.NoError(t, err)
	assert.Equal(t, TimeCapsuleType, result.Type())
	got, _ := GetTime(result)
	assert.Equal(t, 10, got.Hour())
	assert.Equal(t, 30, got.Minute())
}

// --- since / until ---

func TestSince(t *testing.T) {
	past := NewTimeCapsule(time.Now().Add(-5 * time.Second))
	result, err := SinceFunc.Call([]cty.Value{past})
	require.NoError(t, err)
	d, _ := GetDuration(result)
	assert.True(t, d >= 5*time.Second)
	assert.True(t, d < 10*time.Second)
}

func TestUntil(t *testing.T) {
	future := NewTimeCapsule(time.Now().Add(5 * time.Second))
	result, err := UntilFunc.Call([]cty.Value{future})
	require.NoError(t, err)
	d, _ := GetDuration(result)
	assert.True(t, d > 0)
	assert.True(t, d <= 5*time.Second)
}

// --- formattime ---

func TestFormatTime(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := FormatTimeFunc.Call([]cty.Value{
		cty.StringVal("2006-01-02"),
		ts,
	})
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15", result.AsString())
}

func TestFormatTimeRFC3339(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := FormatTimeFunc.Call([]cty.Value{
		cty.StringVal("2006-01-02T15:04:05Z07:00"),
		ts,
	})
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.AsString())
}

func TestFormatTimeNamedFormat(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := FormatTimeFunc.Call([]cty.Value{cty.StringVal("@date"), ts})
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15", result.AsString())
}

func TestFormatTimeNamedFormatRFC3339(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC))
	result, err := FormatTimeFunc.Call([]cty.Value{cty.StringVal("@rfc3339"), ts})
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15T10:30:00Z", result.AsString())
}

// --- strftime / strptime ---

func TestStrftime(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC))
	result, err := StrftimeFunc.Call([]cty.Value{cty.StringVal("%Y-%m-%d"), ts})
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15", result.AsString())
}

func TestStrftimeHourMinute(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC))
	result, err := StrftimeFunc.Call([]cty.Value{cty.StringVal("%H:%M:%S"), ts})
	require.NoError(t, err)
	assert.Equal(t, "10:30:45", result.AsString())
}

func TestStrptime(t *testing.T) {
	result, err := StrptimeFunc.Call([]cty.Value{
		cty.StringVal("%Y-%m-%d"),
		cty.StringVal("2024-01-15"),
	})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 2024, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 15, got.Day())
}

func TestStrptimeWithTimezone(t *testing.T) {
	result, err := StrptimeFunc.Call([]cty.Value{
		cty.StringVal("%Y-%m-%d"),
		cty.StringVal("2024-01-15"),
		cty.StringVal("America/New_York"),
	})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, "America/New_York", got.Location().String())
	assert.Equal(t, 15, got.Day())
}

func TestStrptimeInvalidFormat(t *testing.T) {
	_, err := StrptimeFunc.Call([]cty.Value{
		cty.StringVal("%Q"),
		cty.StringVal("2024-01-15"),
	})
	assert.Error(t, err)
}

// --- fromunix / unix ---

func TestFromUnixSeconds(t *testing.T) {
	result, err := FromUnixFunc.Call([]cty.Value{cty.NumberIntVal(0)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.Unix(0, 0).UTC(), got)
}

func TestFromUnixFractionalSeconds(t *testing.T) {
	result, err := FromUnixFunc.Call([]cty.Value{cty.NumberFloatVal(1.5)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.Unix(1, 500_000_000).UTC(), got)
}

func TestFromUnixUnits(t *testing.T) {
	base := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		n    int64
		unit string
	}{
		{base.Unix(), "s"},
		{base.UnixMilli(), "ms"},
		{base.UnixMicro(), "us"},
		{base.UnixNano(), "ns"},
	}
	for _, tt := range tests {
		result, err := FromUnixFunc.Call([]cty.Value{cty.NumberIntVal(tt.n), cty.StringVal(tt.unit)})
		require.NoError(t, err, "fromunix(%d, %q)", tt.n, tt.unit)
		got, _ := GetTime(result)
		assert.True(t, base.Equal(got), "fromunix(%d, %q): got %v", tt.n, tt.unit, got)
	}
}

func TestFromUnixInvalidUnit(t *testing.T) {
	_, err := FromUnixFunc.Call([]cty.Value{cty.NumberIntVal(0), cty.StringVal("days")})
	assert.Error(t, err)
}

func TestUnixSeconds(t *testing.T) {
	ts := NewTimeCapsule(time.Unix(1705312200, 500_000_000).UTC())
	result, err := UnixFunc.Call([]cty.Value{ts})
	require.NoError(t, err)
	f, _ := result.AsBigFloat().Float64()
	assert.InDelta(t, 1705312200.5, f, 1e-6)
}

func TestUnixUnits(t *testing.T) {
	base := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ts := NewTimeCapsule(base)
	tests := []struct {
		unit string
		want int64
	}{
		{"ms", base.UnixMilli()},
		{"us", base.UnixMicro()},
		{"ns", base.UnixNano()},
	}
	for _, tt := range tests {
		result, err := UnixFunc.Call([]cty.Value{ts, cty.StringVal(tt.unit)})
		require.NoError(t, err)
		got, _ := result.AsBigFloat().Int64()
		assert.Equal(t, tt.want, got, "unix(t, %q)", tt.unit)
	}
}

// --- timezone / intimezone ---

func TestTimezoneNoArgs(t *testing.T) {
	result, err := TimezoneFunc.Call([]cty.Value{})
	require.NoError(t, err)
	assert.Equal(t, cty.String, result.Type())
	// Should be a non-empty string
	assert.NotEmpty(t, result.AsString())
}

func TestTimezoneWithTime(t *testing.T) {
	utc := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	result, err := TimezoneFunc.Call([]cty.Value{utc})
	require.NoError(t, err)
	assert.Equal(t, "UTC", result.AsString())
}

func TestTimezoneNamedZone(t *testing.T) {
	ny, _ := time.LoadLocation("America/New_York")
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, ny))
	result, err := TimezoneFunc.Call([]cty.Value{ts})
	require.NoError(t, err)
	assert.Equal(t, "America/New_York", result.AsString())
}

func TestInTimezone(t *testing.T) {
	utc := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	ts := NewTimeCapsule(utc)
	result, err := InTimezoneFunc.Call([]cty.Value{ts, cty.StringVal("America/New_York")})
	require.NoError(t, err)
	got, _ := GetTime(result)
	// Same instant
	assert.True(t, utc.Equal(got))
	// Different display timezone
	assert.Equal(t, "America/New_York", got.Location().String())
	// 10:00 UTC = 05:00 EST
	assert.Equal(t, 5, got.Hour())
}

func TestInTimezoneInvalidZone(t *testing.T) {
	ts := NewTimeCapsule(time.Now())
	_, err := InTimezoneFunc.Call([]cty.Value{ts, cty.StringVal("Not/ATimezone")})
	assert.Error(t, err)
}

// --- calendar arithmetic ---

func TestAddYears(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	result, err := AddYearsFunc.Call([]cty.Value{ts, cty.NumberIntVal(2)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 2026, got.Year())
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 15, got.Day())
}

func TestAddYearsLeapDay(t *testing.T) {
	// Feb 29 on leap year + 1 year = Feb 28 on non-leap year (Go's AddDate behaviour)
	ts := NewTimeCapsule(time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC))
	result, err := AddYearsFunc.Call([]cty.Value{ts, cty.NumberIntVal(1)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, 2025, got.Year())
	assert.Equal(t, time.March, got.Month()) // Go normalises Feb 29 → Mar 1
	assert.Equal(t, 1, got.Day())
}

func TestAddMonths(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	result, err := AddMonthsFunc.Call([]cty.Value{ts, cty.NumberIntVal(3)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.April, got.Month())
	assert.Equal(t, 15, got.Day())
}

func TestAddDays(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	result, err := AddDaysFunc.Call([]cty.Value{ts, cty.NumberIntVal(20)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.February, got.Month())
	assert.Equal(t, 4, got.Day())
}

func TestAddDaysNegative(t *testing.T) {
	ts := NewTimeCapsule(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC))
	result, err := AddDaysFunc.Call([]cty.Value{ts, cty.NumberIntVal(-5)})
	require.NoError(t, err)
	got, _ := GetTime(result)
	assert.Equal(t, time.January, got.Month())
	assert.Equal(t, 10, got.Day())
}

// --- comparison functions ---

func TestTimeBefore(t *testing.T) {
	t1 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	t2 := NewTimeCapsule(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC))

	result, err := TimeBeforeFunc.Call([]cty.Value{t1, t2})
	require.NoError(t, err)
	assert.True(t, result.True())

	result, err = TimeBeforeFunc.Call([]cty.Value{t2, t1})
	require.NoError(t, err)
	assert.False(t, result.True())

	// Equal times: not before
	result, err = TimeBeforeFunc.Call([]cty.Value{t1, t1})
	require.NoError(t, err)
	assert.False(t, result.True())
}

func TestTimeAfter(t *testing.T) {
	t1 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	t2 := NewTimeCapsule(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC))

	result, err := TimeAfterFunc.Call([]cty.Value{t2, t1})
	require.NoError(t, err)
	assert.True(t, result.True())

	result, err = TimeAfterFunc.Call([]cty.Value{t1, t2})
	require.NoError(t, err)
	assert.False(t, result.True())
}

// --- capsule equality ---

func TestTimeCapsuleEquality(t *testing.T) {
	t1 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))
	t2 := NewTimeCapsule(time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC))
	t3 := NewTimeCapsule(time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC))

	// Same instant — equal
	assert.True(t, t1.Equals(t3).True())
	// Different instants — not equal
	assert.True(t, t1.Equals(t2).False())
}

func TestTimeCapsuleEqualityAcrossTimezones(t *testing.T) {
	utc := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	ny, _ := time.LoadLocation("America/New_York")
	// Same instant expressed in different timezones
	utcVal := NewTimeCapsule(utc)
	nyVal := NewTimeCapsule(utc.In(ny))
	assert.True(t, utcVal.Equals(nyVal).True())
}

// --- resolveFormat / @name aliases ---

func TestResolveFormatPassthrough(t *testing.T) {
	layout, err := resolveFormat("2006-01-02")
	require.NoError(t, err)
	assert.Equal(t, "2006-01-02", layout)
}

func TestResolveFormatNamedAliases(t *testing.T) {
	tests := []struct {
		name   string
		layout string
	}{
		{"@rfc3339", time.RFC3339},
		{"@rfc3339nano", time.RFC3339Nano},
		{"@date", time.DateOnly},
		{"@time", time.TimeOnly},
		{"@datetime", time.DateTime},
		{"@RFC3339", time.RFC3339}, // case-insensitive
	}
	for _, tt := range tests {
		got, err := resolveFormat(tt.name)
		require.NoError(t, err, "resolveFormat(%q)", tt.name)
		assert.Equal(t, tt.layout, got, "resolveFormat(%q)", tt.name)
	}
}

func TestResolveFormatUnknown(t *testing.T) {
	_, err := resolveFormat("@bogus")
	assert.Error(t, err)
}
