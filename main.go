package main

import (
	"github.com/fever-ch/http-ping/cmd"
	"github.com/fever-ch/http-ping/stats"
	"os"
)

func main() {
	i := -int64(1) << 63
	println(i)
	m := stats.Measure(i)
	println(int64(m))
	cmd.Execute()
	os.Exit(0)
}
