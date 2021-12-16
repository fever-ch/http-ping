package app

import (
	"fmt"
	"github.com/fever-ch/http-ping/stats"
	"time"
)

// HTTPMeasure is the out of a measurement done as an HTTP ping
type HTTPMeasure struct {
	Proto        string
	Duration     stats.Measure
	StatusCode   int
	Bytes        int64
	InBytes      int64
	OutBytes     int64
	SocketReused bool
	Compressed   bool
	RemoteAddr   string
	TLSEnabled   bool
	TLSVersion   string

	DNSDuration  stats.Measure
	TCPHandshake stats.Measure
	TLSDuration  stats.Measure
	ConnDuration stats.Measure
	ReqDuration  stats.Measure
	RespDuration stats.Measure
	Wait         stats.Measure
	IsFailure    bool
	FailureCause string
}

// Pinger is responsible for actually doing the HTTP pings
type Pinger struct {
	client *WebClient
	config *Config
}

// NewPinger builds a new Pinger
func NewPinger(config *Config) (*Pinger, error) {

	pinger := Pinger{}

	pinger.config = config

	client, err := NewWebClient(config)

	if err != nil {
		return nil, fmt.Errorf("%s (%s)", err, config.IPProtocol)
	}

	pinger.client = client

	return &pinger, nil
}

// Ping actually does the pinging specified in config
func (pinger *Pinger) Ping() <-chan *HTTPMeasure {
	measures := make(chan *HTTPMeasure)
	go func() {

		if !pinger.config.DisableKeepAlive {
			pinger.client.DoMeasure()
			time.Sleep(pinger.config.Interval)
		}

		for a := int64(0); a < pinger.config.Count; a++ {
			measures <- pinger.client.DoMeasure()
			time.Sleep(pinger.config.Interval)
		}

		close(measures)
	}()
	return measures
}
