package app

import "runtime/debug"

var (
	// Version of the application
	Version = "undef"

	// ProjectURL is the URL of the project
	ProjectURL = "https://github.com/fever-ch/http-ping"
)

func init() {
	buildInfo, _ := debug.ReadBuildInfo()
	Version = buildInfo.Main.Version
}
