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

	_, _ = fmt.Fprintf(stdout, "HTTP-PING %s %s\n\n", pinger.client.url.String(), config.Method)

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
						if config.LogLevel >= 1 {
							_, _ = fmt.Fprintf(stdout, "%8d: %s, code=%d, size=%d bytes, time=%.1f ms\n", attempts, measure.RemoteAddr, measure.StatusCode, measure.Bytes, float64(measure.Duration.Nanoseconds())/1e6)
						}
						if config.LogLevel == 2 {

							_, _ = fmt.Fprintf(stdout, "          [network i/o: %d bytes read, %d bytes written]\n", measure.InBytes, measure.OutBytes)

							z := measureEntry{
								label:    "request and response",
								duration: measure.Duration,
								children: []*measureEntry{
									{label: "connection setup", duration: measure.ConnDuration,
										children: []*measureEntry{
											{label: "DNS resolution", duration: measure.DNSDuration},
											{label: "TCP handshake", duration: measure.TCPHandshake},
											{label: "TLS handshake", duration: measure.TLSDuration},
										}},
									{label: "request sending", duration: measure.ReqDuration},
									{label: "wait", duration: measure.Wait},
									{label: "response ingestion", duration: measure.RespDuration},
								},
							}

							//if no TLS
							//z.children[0].children = z.children[0].children[0:2]

							l := makeVisitList(&z)

							for i, e := range l {
								pipes := make([]string, e.depth)
								for j := 0; j < e.depth; j++ {
									if i+1 >= len(l) || l[i+1].depth-1 < j {
										pipes[j] = "└"
									} else {
										pipes[j] = "│"
									}

								}
								_, _ = fmt.Fprintf(stdout, "          ")
								for i := 0; i < e.depth; i++ {
									_, _ = fmt.Fprintf(stdout, "          %s ", pipes[i])
								}

								space := ""

								if e.measureEntry.children != nil && len(e.measureEntry.children) > 0 {
									space = "┬ "
								}

								_, _ = fmt.Fprintf(stdout, "%6.1f ms %s%s\n", float64(e.measureEntry.duration.Nanoseconds())/1e6, space, e.measureEntry.label)
							}
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

	fmt.Printf("\n--- %s ping statistics ---\n", pinger.client.url.String())
	var lossRate = float64(0)
	if attempts > 0 {
		lossRate = float64(100*failures) / float64(attempts)
	}

	_, _ = fmt.Fprintf(stdout, "%d requests sent, %d answers received, %.1f%% loss\n", attempts, attempts-failures, lossRate)

	if len(latencies) > 0 {
		_, _ = fmt.Fprintf(stdout, "%s\n", stats.PingStatsFromLatencies(latencies).String())
	}

}

type measureEntry struct {
	label    string
	duration time.Duration
	children []*measureEntry
}

type measureEntryVisit struct {
	measureEntry *measureEntry
	depth        int
}

func makeVisitList(root *measureEntry) []measureEntryVisit {
	var visits []measureEntryVisit

	var visit func(entry *measureEntry, depth int)

	visit = func(entry *measureEntry, depth int) {
		visits = append(visits, measureEntryVisit{entry, depth})

		for _, e := range entry.children {

			visit(e, depth+1)
		}

	}

	visit(root, 0)
	return visits
}
