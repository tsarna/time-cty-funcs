package timecty

import _ "embed"

//go:embed externs.cty
var externsCty []byte

// ExternsFilename is the name reported for the embedded declarations in
// diagnostics.
const ExternsFilename = "time-cty-funcs/externs.cty"

// Externs returns the functy `//functy:extern` declarations for the functions
// GetTimeFunctions provides whose real signature their cty metadata cannot express.
//
// cty can only make a function's *trailing* parameters optional, so an optional
// argument has to be faked with a variadic — which erases its name, its type, and its
// default: `fromunix(n, unit = "s")` reflects as `fromunix(n, ...args)`. And a cty
// function has one signature, so a function whose argument shapes differ per call
// (`parsetime(s)` vs `parsetime(format, s)`) or whose return type depends on what it
// was handed (`timeadd`) cannot be described at all. The declarations here say what
// each really accepts, so that help(), generated documentation, and editor tooling
// can show it.
//
// The bytes are opaque to this package: it does not import functy, and nothing here
// parses them. A functy host registers them:
//
//	parser.RegisterExterns(timecty.Externs(), timecty.ExternsFilename)
//
// The other functions this package provides are deliberately absent — their cty
// metadata is complete (fixed arity, concrete parameter types, a static return type),
// so an extern would only be a second place for the same facts to drift.
func Externs() []byte { return externsCty }
