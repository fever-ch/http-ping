package app

import (
	"fmt"
	"time"
)

// Answer is the out of a measurement done as an HTTP ping
type Answer struct {
	Proto        string
	Duration     time.Duration
	StatusCode   int
	Bytes        int64
	InBytes      int64
	OutBytes     int64
	SocketReused bool

	IsFailure    bool
	FailureCause string
}

func (a *Answer) String() string {
	return fmt.Sprintf("code=%d size=%d conn-reused=%t time=%.3f ms", a.StatusCode, a.Bytes, a.SocketReused, float64(a.Duration.Nanoseconds())/1e6)
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
func (pinger *Pinger) Ping() <-chan *Answer {
	measures := make(chan *Answer)
	go func() {
		pinger.client.DoMeasure()

		for a := int64(0); a < pinger.config.Count; a++ {
			measures <- pinger.client.DoMeasure()
			time.Sleep(pinger.config.Interval)
		}

		close(measures)
	}()
	return measures
}
