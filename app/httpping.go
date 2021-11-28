package app

import (
	"fmt"
	"github.com/fever-ch/http-ping/stats"
	"github.com/fever-ch/http-ping/util"
	"net/url"
	"os"
	"time"
)

// HttpPing actually does the pinging specified in config
func HttpPing(config Config) {

	u, _ := url.Parse(config.Target())

	client, err := NewWebClient(config)

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s (%s)\n", err, config.IpProtocol())
		os.Exit(1)
	}

	fmt.Printf("HTTP-PING %s (%s) %s\n", u.String(), client.ipAddr.IP, config.Method())

	var latencies []time.Duration

	sh := util.NewSignalHandler(os.Interrupt)

	_ = client.DoConnection()
	sh.Sleep(config.Interval())

	attempts, failures := 0, 0

	for a := int64(0); a < config.Count() && !sh.Triggered(); a++ {
		attempts++
		if measure, err := client.DoMeasure(); err == nil {
			if config.LogLevel() >= 1 {
				fmt.Printf("%4d: %s\n", a, measure)
			}
			latencies = append(latencies, measure.Duration)
		} else {
			failures++
			if config.LogLevel() >= 1 {
				fmt.Printf("%4d: Request timeout\n", a)
			}
		}
		if a < config.Count() {
			sh.Sleep(config.Interval())
		}
	}

	fmt.Printf("\n--- %s (%s) ping statistics ---\n", u.String(), client.ipAddr.IP)
	var lossRate = float64(0)
	if len(latencies) > 0 {
		lossRate = float64(100*failures) / float64(attempts)
	}

	fmt.Printf("%d requests sent, %d answers received, %.1f%% loss\n", attempts, attempts-failures, lossRate)

	if len(latencies) > 0 {
		fmt.Printf("%s\n", stats.PingStatsFromLatencies(latencies).String())
	}
}
