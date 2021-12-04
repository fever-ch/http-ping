package cmd

import (
	"github.com/fever-ch/http-ping/app"
	"regexp"
	"time"
)

type cmdConfig struct {
	target string

	ipv4 bool
	ipv6 bool

	head bool

	userAgent string

	connTarget string

	fullConnection bool

	wait time.Duration

	count int64

	interval time.Duration

	verbose bool

	quiet bool

	noCheckCertificate bool

	cookies []string

	parameters []string
}

func (c *cmdConfig) LogLevel() int8 {
	if c.verbose {
		return 2
	}
	if c.quiet {
		return 0
	}
	return 1
}

func (c *cmdConfig) Method() string {
	if c.head {
		return "HEAD"
	}
	return "GET"
}

func splitPair(str string) (string, string) {
	r := regexp.MustCompile("^([^:]*):(.*)$")
	e := r.FindStringSubmatch(str)
	if len(e) == 3 {
		return e[1], e[2]
	}
	return "", ""
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

func (c *cmdConfig) IPProtocol() string {
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

func (c *cmdConfig) ConnTarget() *string {
	if c.connTarget == "" {
		return nil
	}
	return &c.connTarget
}

func (c *cmdConfig) NoCheckCertificate() bool {
	return c.noCheckCertificate
}

func (c *cmdConfig) Cookies() []app.Cookie {
	var cookies []app.Cookie
	for _, cookie := range c.cookies {
		n, v := splitPair(cookie)
		if n != "" {
			cookies = append(cookies, app.Cookie{Name: n, Value: v})
		}
	}
	return cookies
}

func (c *cmdConfig) Parameters() []app.Parameter {
	var parameters []app.Parameter
	for _, parameter := range c.parameters {
		n, v := splitPair(parameter)
		if n != "" {
			parameters = append(parameters, app.Parameter{Name: n, Value: v})
		}
	}
	return parameters
}
