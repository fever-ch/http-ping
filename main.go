package main

import (
	"github.com/fever-ch/http-ping/cmd"
	"os"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}
