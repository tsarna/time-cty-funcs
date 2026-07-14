package timecty

import (
	"embed"
	"path"
)

//go:embed externs/*.cty
var externsFS embed.FS

// Externs returns the functy `//functy:extern` declarations for the functions
// GetTimeFunctions provides whose real signature their cty metadata cannot express,
// keyed by a filename to report in diagnostics.
//
// cty can only make a function's *trailing* parameters optional, so an optional
// argument has to be faked with a variadic — which erases its name, its type, and its
// default: `time::from_unix(n, unit = "s")` reflects as `from_unix(n, ...args)`. And a
// cty function has one signature, so a function whose argument shapes differ per call
// (`time::parse(s)` vs `time::parse(format, s)`) or whose parameters are a union
// (`dns::parse_zone_serial` takes a number *or* a string) cannot be described at all.
// These declarations say what each really accepts, so that help(), generated
// documentation, and editor tooling can show it.
//
// There is one file per namespace, because a functy source declares at most one:
// `time`, `duration`, `dns`, and a global file for the un-namespaced `duration()`
// constructor. A host registers them all:
//
//	for name, src := range timecty.Externs() {
//	    parser.RegisterExterns(src, name)
//	}
//
// The bytes are opaque to this package: it does not import functy, and nothing here
// parses them. The functions with no declaration are deliberately undeclared — their
// cty metadata is complete, so an extern would only be a second place for the same
// facts to drift.
func Externs() map[string][]byte {
	entries, err := externsFS.ReadDir("externs")
	if err != nil {
		// Unreachable: the directory is embedded at build time.
		panic("timecty: embedded externs unreadable: " + err.Error())
	}

	out := make(map[string][]byte, len(entries))
	for _, e := range entries {
		name := path.Join("externs", e.Name())
		src, err := externsFS.ReadFile(name)
		if err != nil {
			panic("timecty: embedded extern unreadable: " + err.Error())
		}
		out[path.Join("time-cty-funcs", name)] = src
	}
	return out
}
