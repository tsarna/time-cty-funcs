package timecty

import "github.com/zclconf/go-cty/cty/function"

// GetTimeFunctions returns this package's cty functions, keyed by the name they are
// callable under.
//
// The names are namespaced — `time::add`, `duration::truncate`, `dns::next_zone_serial`.
// HCL parses `a::b(x)` natively and resolves it as a single flat map key, so a namespace
// is a naming convention rather than a containment relationship: these are ordinary map
// entries whose keys happen to contain `::`.
//
// Two rules, both HashiCorp's for provider-defined functions, and both worth stating
// because they are why the names look the way they do:
//
//   - The leaf name does not repeat the namespace. `time::add`, not `time::timeadd`.
//   - Namespaced functions use underscores. HCL's *built-in* functions run words
//     together (`formatdate`, `jsonencode`) for historical reasons; once a namespace
//     does the grouping, the leaf name gets longer and more descriptive, and run-on
//     stops being readable — `next_zone_serial` over `nextzoneserial`.
//
// `duration` keeps a bare, un-namespaced name: it is the type constructor, and reads as
// one — `duration("1h30m")`, like `tostring(x)`. A bare name and a namespace of the same
// name coexist without conflict, since they are simply different map keys.
//
// There is deliberately no `timeadd`. That name belongs to cty's own
// stdlib.TimeAddFunc, the string-in/string-out function HCL has always had; a host that
// wants it registers it directly. `time::add` carries no compatibility burden as a
// result, which is why every one of its forms returns a time.
func GetTimeFunctions() map[string]function.Function {
	return map[string]function.Function{
		// Constructing and parsing times
		"time::now":       NowFunc,
		"time::parse":     ParseTimeFunc,
		"time::from_unix": FromUnixFunc,
		"time::strptime":  StrptimeFunc,

		// Rendering times
		"time::format":   FormatTimeFunc,
		"time::strftime": StrftimeFunc,

		// Time arithmetic
		"time::add":        TimeAddFunc,
		"time::sub":        TimeSubFunc,
		"time::since":      SinceFunc,
		"time::until":      UntilFunc,
		"time::add_years":  AddYearsFunc,
		"time::add_months": AddMonthsFunc,
		"time::add_days":   AddDaysFunc,

		// Decomposing times
		"time::to_unix": UnixFunc,
		"time::zone":    TimezoneFunc,
		"time::in_zone": InTimezoneFunc,

		// Comparing times
		"time::before": TimeBeforeFunc,
		"time::after":  TimeAfterFunc,

		// Constructing durations. The bare name is the type constructor; see above.
		"duration": DurationFunc,

		// Rendering durations
		"duration::format": FormatDurationFunc,

		// Duration arithmetic
		"duration::add":      DurationAddFunc,
		"duration::sub":      DurationSubFunc,
		"duration::mul":      DurationMulFunc,
		"duration::div":      DurationDivFunc,
		"duration::truncate": DurationTruncateFunc,
		"duration::round":    DurationRoundFunc,
		"duration::abs":      AbsDurationFunc,

		// Comparing durations
		"duration::lt": DurationLtFunc,
		"duration::gt": DurationGtFunc,

		// DNS zone serials. Their own namespace: they are not time functions, and
		// `nextzoneserial`/`parsezoneserial` were only ever named that way because they
		// had nowhere else to live.
		"dns::next_zone_serial":  NextZoneSerialFunc,
		"dns::parse_zone_serial": ParseZoneSerialFunc,
	}
}
