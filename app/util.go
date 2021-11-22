package app

import "runtime/debug"

var (
	// Version of the application
	Version = "undef"

	// ProjectUrl is the URL of the project
	ProjectUrl = "https://github.com/fever-ch/http-ping"
)

func init() {
	buildinfo, _ := debug.ReadBuildInfo()
	Version = buildinfo.Main.Version
}
