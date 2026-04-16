package timecty

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/zclconf/go-cty/cty"
)

// Timestamp wraps a time.Time so it can implement the rich-cty-types
// Stringable and Gettable interfaces. Embedding forwards time.Time's methods
// (Format, Year, Equal, etc.) so callers can use them directly.
type Timestamp struct {
	time.Time
}

// Duration wraps a time.Duration so it can implement the rich-cty-types
// Stringable and Gettable interfaces.
type Duration struct {
	time.Duration
}

// TimeCapsuleType is a cty capsule type wrapping Timestamp.
// Supports equality (==, !=) via Equals/RawEquals.
// Note: ordering operators (<, >, etc.) are not available for capsule types
// in go-cty — use timesub() and compare the resulting duration instead.
var TimeCapsuleType = cty.CapsuleWithOps("time", reflect.TypeOf(Timestamp{}), &cty.CapsuleOps{
	// Equals uses time.Time.Equal so that two instants in different timezones
	// that represent the same moment compare as equal.
	Equals: func(a, b any) cty.Value {
		ta := a.(*Timestamp)
		tb := b.(*Timestamp)
		return cty.BoolVal(ta.Equal(tb.Time))
	},
	RawEquals: func(a, b any) bool {
		ta := a.(*Timestamp)
		tb := b.(*Timestamp)
		return ta.Equal(tb.Time)
	},
	GoString: func(val any) string {
		return fmt.Sprintf("time(%q)", val.(*Timestamp).Format(time.RFC3339Nano))
	},
	TypeGoString: func(_ reflect.Type) string {
		return "time"
	},
})

// DurationCapsuleType is a cty capsule type wrapping Duration.
// Supports equality (==, !=) via Equals/RawEquals.
// Note: ordering operators (<, >, etc.) are not available for capsule types
// in go-cty — use get(d, unit) (via rich-cty-types) to extract a numeric
// value and compare that instead.
var DurationCapsuleType = cty.CapsuleWithOps("duration", reflect.TypeOf(Duration{}), &cty.CapsuleOps{
	Equals: func(a, b any) cty.Value {
		da := a.(*Duration)
		db := b.(*Duration)
		return cty.BoolVal(da.Duration == db.Duration)
	},
	RawEquals: func(a, b any) bool {
		da := a.(*Duration)
		db := b.(*Duration)
		return da.Duration == db.Duration
	},
	GoString: func(val any) string {
		return fmt.Sprintf("duration(%q)", val.(*Duration).Duration.String())
	},
	TypeGoString: func(_ reflect.Type) string {
		return "duration"
	},
})

// NewTimeCapsule wraps a time.Time in a cty capsule value.
func NewTimeCapsule(t time.Time) cty.Value {
	return cty.CapsuleVal(TimeCapsuleType, &Timestamp{t})
}

// GetTime extracts a time.Time from a cty capsule value.
// Returns an error if the value is not a TimeCapsuleType.
func GetTime(val cty.Value) (time.Time, error) {
	if val.Type() != TimeCapsuleType {
		return time.Time{}, fmt.Errorf("expected time capsule, got %s", val.Type().FriendlyName())
	}
	return val.EncapsulatedValue().(*Timestamp).Time, nil
}

// NewDurationCapsule wraps a time.Duration in a cty capsule value.
func NewDurationCapsule(d time.Duration) cty.Value {
	return cty.CapsuleVal(DurationCapsuleType, &Duration{d})
}

// GetDuration extracts a time.Duration from a cty capsule value.
// Returns an error if the value is not a DurationCapsuleType.
func GetDuration(val cty.Value) (time.Duration, error) {
	if val.Type() != DurationCapsuleType {
		return 0, fmt.Errorf("expected duration capsule, got %s", val.Type().FriendlyName())
	}
	return val.EncapsulatedValue().(*Duration).Duration, nil
}

// --- rich-cty-types: Stringable ---

// ToString formats the timestamp as RFC3339Nano. This is lossless for times
// with sub-second precision and identical to RFC3339 for whole-second times.
func (t *Timestamp) ToString(_ context.Context) (string, error) {
	return t.Format(time.RFC3339Nano), nil
}

// ToString formats the duration using Go's default duration syntax (e.g. "1h30m5s").
// This matches the default output of formatduration().
func (d *Duration) ToString(_ context.Context) (string, error) {
	return d.Duration.String(), nil
}

// --- rich-cty-types: Gettable ---

// Get extracts a named calendar field from the timestamp in its stored timezone.
// Valid parts: year, month, day, hour, minute, second, nanosecond, weekday
// (0=Sunday), yearday, isoweek, isoyear.
func (t *Timestamp) Get(_ context.Context, args []cty.Value) (cty.Value, error) {
	if len(args) == 0 {
		return cty.NilVal, fmt.Errorf("time get: part argument required")
	}
	if args[0].Type() != cty.String {
		return cty.NilVal, fmt.Errorf("time get: part argument must be a string")
	}
	switch args[0].AsString() {
	case "year":
		return cty.NumberIntVal(int64(t.Year())), nil
	case "month":
		return cty.NumberIntVal(int64(t.Month())), nil
	case "day":
		return cty.NumberIntVal(int64(t.Day())), nil
	case "hour":
		return cty.NumberIntVal(int64(t.Hour())), nil
	case "minute":
		return cty.NumberIntVal(int64(t.Minute())), nil
	case "second":
		return cty.NumberIntVal(int64(t.Second())), nil
	case "nanosecond":
		return cty.NumberIntVal(int64(t.Nanosecond())), nil
	case "weekday":
		return cty.NumberIntVal(int64(t.Weekday())), nil
	case "yearday":
		return cty.NumberIntVal(int64(t.YearDay())), nil
	case "isoweek":
		_, week := t.ISOWeek()
		return cty.NumberIntVal(int64(week)), nil
	case "isoyear":
		year, _ := t.ISOWeek()
		return cty.NumberIntVal(int64(year)), nil
	default:
		return cty.NilVal, fmt.Errorf("time get: unknown part %q; valid parts: year, month, day, hour, minute, second, nanosecond, weekday, yearday, isoweek, isoyear", args[0].AsString())
	}
}

// Get extracts the duration expressed in the given unit.
// "h", "m", "s" return floats; "ms", "us", "ns" return integers.
func (d *Duration) Get(_ context.Context, args []cty.Value) (cty.Value, error) {
	if len(args) == 0 {
		return cty.NilVal, fmt.Errorf("duration get: unit argument required")
	}
	if args[0].Type() != cty.String {
		return cty.NilVal, fmt.Errorf("duration get: unit argument must be a string")
	}
	switch args[0].AsString() {
	case "h":
		return cty.NumberFloatVal(d.Hours()), nil
	case "m":
		return cty.NumberFloatVal(d.Minutes()), nil
	case "s":
		return cty.NumberFloatVal(d.Seconds()), nil
	case "ms":
		return cty.NumberIntVal(d.Milliseconds()), nil
	case "us":
		return cty.NumberIntVal(d.Microseconds()), nil
	case "ns":
		return cty.NumberIntVal(d.Nanoseconds()), nil
	default:
		return cty.NilVal, fmt.Errorf("duration get: unknown unit %q; valid units: h, m, s, ms, us, ns", args[0].AsString())
	}
}
