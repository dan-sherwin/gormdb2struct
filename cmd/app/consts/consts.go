// Package consts stores application-level build and naming constants.
package consts

// APPNAME is the runtime CLI name.
const APPNAME = "gormdb2struct"

// Version is the application version and is intended to be injected at build time.
var Version = "dev"

// Commit remains for ldflags compatibility with release tooling.
var Commit = ""

// BuildDate remains for ldflags compatibility with release tooling.
var BuildDate = ""
