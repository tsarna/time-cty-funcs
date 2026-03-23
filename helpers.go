package timecty

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	isoduration "github.com/sosodev/duration"
	"github.com/zclconf/go-cty/cty"
)

// namedFormats maps @name aliases to Go reference-time layout strings.
var namedFormats = map[string]string{
	"ansic":       time.ANSIC,
	"unixdate":    time.UnixDate,
	"rubydate":    time.RubyDate,
	"rfc822":      time.RFC822,
	"rfc822z":     time.RFC822Z,
	"rfc850":      time.RFC850,
	"rfc1123":     time.RFC1123,
	"rfc1123z":    time.RFC1123Z,
	"rfc3339":     time.RFC3339,
	"rfc3339nano": time.RFC3339Nano,
	"kitchen":     time.Kitchen,
	"stamp":       time.Stamp,
	"stampmilli":  time.StampMilli,
	"stampmicro":  time.StampMicro,
	"stampnano":   time.StampNano,
	"datetime":    time.DateTime,
	"date":        time.DateOnly,
	"time":        time.TimeOnly,
}

// resolveFormat resolves an @-prefixed named format to its Go layout string.
// Strings not starting with @ are returned unchanged.
func resolveFormat(s string) (string, error) {
	if !strings.HasPrefix(s, "@") {
		return s, nil
	}
	name := strings.ToLower(s[1:])
	if layout, ok := namedFormats[name]; ok {
		return layout, nil
	}
	return "", fmt.Errorf("unknown named format %q; valid names: @ansic, @unixdate, @rubydate, @rfc822, @rfc822z, @rfc850, @rfc1123, @rfc1123z, @rfc3339, @rfc3339nano, @kitchen, @stamp, @stampmilli, @stampmicro, @stampnano, @datetime, @date, @time", s)
}

var durationUnits = map[string]time.Duration{
	"h":  time.Hour,
	"m":  time.Minute,
	"s":  time.Second,
	"ms": time.Millisecond,
	"us": time.Microsecond,
	"ns": time.Nanosecond,
}

// parseDurationString parses a Go-format or ISO 8601 duration string.
// Calendar durations (years, months) are rejected.
func parseDurationString(s string) (cty.Value, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "P") || strings.HasPrefix(s, "-P") {
		d, err := isoduration.Parse(strings.TrimPrefix(s, "-"))
		if err != nil {
			return cty.NilVal, fmt.Errorf("invalid ISO 8601 duration %q: %s", s, err)
		}
		if d.Years != 0 || d.Months != 0 {
			return cty.NilVal, fmt.Errorf("calendar durations with years or months cannot be represented as a fixed duration; use addyears() or addmonths() instead")
		}
		td := d.ToTimeDuration()
		if strings.HasPrefix(s, "-") {
			td = -td
		}
		return NewDurationCapsule(td), nil
	}
	td, err := time.ParseDuration(s)
	if err != nil {
		return cty.NilVal, fmt.Errorf("invalid duration %q: expected ISO 8601 (e.g. \"PT5M\") or Go format (e.g. \"5m30s\"): %s", s, err)
	}
	return NewDurationCapsule(td), nil
}

// durationFromNumber constructs a duration from a number and a unit string.
func durationFromNumber(n float64, unit string) (cty.Value, error) {
	factor, ok := durationUnits[unit]
	if !ok {
		return cty.NilVal, fmt.Errorf("unknown duration unit %q; valid units: h, m, s, ms, us, ns", unit)
	}
	return NewDurationCapsule(time.Duration(n * float64(factor))), nil
}

// durationToISO8601 formats a time.Duration as an ISO 8601 duration string (P-notation).
func durationToISO8601(d time.Duration) string {
	if d == 0 {
		return "PT0S"
	}

	prefix := ""
	if d < 0 {
		prefix = "-"
		d = -d
	}

	hours := int64(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	minutes := int64(d / time.Minute)
	d -= time.Duration(minutes) * time.Minute

	totalNs := d.Nanoseconds()
	seconds := totalNs / 1_000_000_000
	fracNs := totalNs % 1_000_000_000

	var b strings.Builder
	b.WriteString(prefix + "PT")
	if hours > 0 {
		fmt.Fprintf(&b, "%dH", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&b, "%dM", minutes)
	}
	if seconds > 0 || fracNs > 0 || (hours == 0 && minutes == 0) {
		if fracNs == 0 {
			fmt.Fprintf(&b, "%dS", seconds)
		} else {
			fracStr := strings.TrimRight(fmt.Sprintf("%09d", fracNs), "0")
			fmt.Fprintf(&b, "%d.%sS", seconds, fracStr)
		}
	}
	return b.String()
}

// parseSerialArg parses a zone serial from a cty.Number or cty.String value.
func parseSerialArg(v cty.Value, funcName string) (int64, error) {
	switch v.Type() {
	case cty.Number:
		n, _ := v.AsBigFloat().Int64()
		return n, nil
	case cty.String:
		n, err := strconv.ParseInt(v.AsString(), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%s: invalid serial %q: %s", funcName, v.AsString(), err)
		}
		return n, nil
	default:
		return 0, fmt.Errorf("%s: serial must be a number or string, got %s", funcName, v.Type().FriendlyName())
	}
}
