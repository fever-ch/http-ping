package app

import "runtime/debug"

var (
	// Version of the application
	Version = "undef"

	// ProjectURL is the URL of the project
	ProjectURL = "https://github.com/fever-ch/http-ping"
)

func init() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		Version = buildInfo.Main.Version
	}
}
