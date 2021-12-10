package app

import (
	"time"
)

type pair struct {
	Name  string
	Value string
}

// Cookie is a data structure which represents the basic info about a cookie (Name and Value)
type Cookie pair

// Header is a data structure which represents the basic info about a HTTP header (Name and Value)
type Header pair

// Parameter is a data structure which represents a request parameter (Name and Value)
type Parameter pair

// Config defines the multiple parameters which can be sent to HTTPPing
type Config struct {
	IPProtocol         string
	Interval           time.Duration
	Count              int64
	Target             string
	Method             string
	UserAgent          string
	Wait               time.Duration
	DisableKeepAlive   bool
	LogLevel           int8
	ConnTarget         string
	NoCheckCertificate bool
	Cookies            []Cookie
	Headers            []Header
	Parameters         []Parameter
	IgnoreServerErrors bool
	ExtraParam         bool
	DisableCompression bool
	AudibleBell        bool
}
