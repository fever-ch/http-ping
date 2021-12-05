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

// Parameter is a data structure which represents the basic info about a cookie (Name and Value)
type Parameter pair

// Config defines the multiple parameters which can be sent to HTTPPing
type Config interface {
	IPProtocol() string
	Interval() time.Duration
	Count() int64
	Target() string
	Method() string
	UserAgent() string
	Wait() time.Duration
	KeepAlive() bool
	LogLevel() int8
	ConnTarget() *string
	NoCheckCertificate() bool
	Cookies() []Cookie
	Parameters() []Parameter
	IgnoreServerErrors() bool
}
