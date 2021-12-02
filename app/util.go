package app

import "runtime/debug"

var (
	// Version of the application
	Version = "undef"

	// ProjectURL is the URL of the project
	ProjectURL = "https://github.com/fever-ch/http-ping"
)

func init() {
	buildinfo, _ := debug.ReadBuildInfo()
	Version = buildinfo.Main.Version
}
