package cmd

import (
	"regexp"
	"time"
)

type cmdConfig struct {
	target string

	ipv4 bool
	ipv6 bool

	head bool

	userAgent string

	fullConnection bool

	wait time.Duration

	count int64

	interval time.Duration
}

func (c *cmdConfig) Method() string {
	if c.head {
		return "HEAD"
	}
	return "GET"
}

func (c *cmdConfig) Target() string {
	if a, e := regexp.MatchString("^https?://", c.target); e == nil && a {
		return c.target
	}
	return "https://" + c.target
}

func (c *cmdConfig) Interval() time.Duration {
	return c.interval
}

func (c *cmdConfig) Wait() time.Duration {
	return c.wait
}
func (c *cmdConfig) Count() int64 {
	return c.count
}

func (c *cmdConfig) IpProtocol() string {
	if c.ipv4 {
		return "ip4"
	} else if c.ipv6 {
		return "ip6"
	} else {
		return "ip"
	}
}

func (c *cmdConfig) KeepAlive() bool {
	return !c.fullConnection
}

func (c *cmdConfig) UserAgent() string {
	return c.userAgent
}
