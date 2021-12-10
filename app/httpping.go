package app

import (
	"fmt"
	"github.com/fever-ch/http-ping/stats"
	"io"
	"os"
	"os/signal"
	"time"
)

// HTTPPing actually does the pinging specified in config
func HTTPPing(config *Config, stdout io.Writer, stderr io.Writer) {

	ic := make(chan os.Signal, 1)

	signal.Notify(ic, os.Interrupt)

	pinger, err := NewPinger(config)

	if err != nil {
		_, _ = fmt.Fprintf(stdout, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	ch := pinger.Ping()

	_, _ = fmt.Fprintf(stdout, "HTTP-PING %s (%s) %s\n", pinger.client.url.String(), pinger.client.connTarget, config.Method)

	var latencies []time.Duration
	attempts, failures := 0, 0

	var loop = true
	for loop {
		select {
		case measure := <-ch:
			{
				if measure == nil {
					loop = false
				} else {
					if !measure.IsFailure {
						if config.LogLevel == 1 {
							_, _ = fmt.Fprintf(stdout, "%4d: code=%d size=%d time=%.3f ms\n", attempts, measure.StatusCode, measure.Bytes, float64(measure.Duration.Nanoseconds())/1e6)
						} else if config.LogLevel == 2 {
							_, _ = fmt.Fprintf(stdout, "%4d: code=%d proto=%s conn-reused=%t compressed=%t size=%d in=%d out=%d time=%.3f ms\n", attempts, measure.StatusCode, measure.Proto, measure.SocketReused, measure.Compressed, measure.Bytes, measure.InBytes, measure.OutBytes, float64(measure.Duration.Nanoseconds())/1e6)
						}
						latencies = append(latencies, measure.Duration)
						if config.AudibleBell {
							_, _ = fmt.Fprintf(stdout, "\a")
						}
					} else {
						if config.LogLevel >= 1 {
							_, _ = fmt.Fprintf(stdout, "%4d: Error: %s\n", attempts, measure.FailureCause)
						}
						failures++
					}
					attempts++
				}
			}
		case <-ic:
			{
				loop = false
			}
		}
	}

	fmt.Printf("\n--- %s (%s) ping statistics ---\n", pinger.client.url.String(), pinger.client.connTarget)
	var lossRate = float64(0)
	if attempts > 0 {
		lossRate = float64(100*failures) / float64(attempts)
	}

	_, _ = fmt.Fprintf(stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, attempts-failures, lossRate)

	if len(latencies) > 0 {
		_, _ = fmt.Fprintf(stdout, "%s\n", stats.PingStatsFromLatencies(latencies).String())
	}

}
