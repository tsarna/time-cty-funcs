package timecty

import "github.com/zclconf/go-cty/cty/function"

// GetTimeFunctions returns all time-related cty functions for registration in an eval context.
// The "timeadd" entry supersedes the stdlib version, adding capsule-type support while
// remaining backward-compatible with the original (string, string) form.
func GetTimeFunctions() map[string]function.Function {
	return map[string]function.Function{
		// Time creation and parsing
		"now":       NowFunc,
		"parsetime": ParseTimeFunc,
		"fromunix":  FromUnixFunc,
		"strptime":  StrptimeFunc,
		// Time formatting
		"formattime": FormatTimeFunc,
		"strftime":   StrftimeFunc,
		// Time arithmetic
		"timeadd":   TimeAddFunc,
		"timesub":   TimeSubFunc,
		"since":     SinceFunc,
		"until":     UntilFunc,
		"addyears":  AddYearsFunc,
		"addmonths": AddMonthsFunc,
		"adddays":   AddDaysFunc,
		// Time decomposition
		"unix":       UnixFunc,
		"timezone":   TimezoneFunc,
		"intimezone": InTimezoneFunc,
		// Time comparison
		"timebefore": TimeBeforeFunc,
		"timeafter":  TimeAfterFunc,
		// Duration creation and parsing
		"duration": DurationFunc,
		// Duration formatting
		"formatduration": FormatDurationFunc,
		// Duration arithmetic
		"durationadd":      DurationAddFunc,
		"durationsub":      DurationSubFunc,
		"durationmul":      DurationMulFunc,
		"durationdiv":      DurationDivFunc,
		"durationtruncate": DurationTruncateFunc,
		"durationround":    DurationRoundFunc,
		"absduration":      AbsDurationFunc,
		// Duration comparison
		"durationlt": DurationLtFunc,
		"durationgt": DurationGtFunc,
		// DNS zone serials
		"nextzoneserial":  NextZoneSerialFunc,
		"parsezoneserial": ParseZoneSerialFunc,
	}
}
