package app

import (
	"time"
)

// Config defines the multiple parameters which can be sent to HttpPing
type Config interface {
	IpProtocol() string
	Interval() time.Duration
	Count() int64
	Target() string
	Method() string
	UserAgent() string
	Wait() time.Duration
	KeepAlive() bool
	LogLevel() int8
	ConnTarget() *string
}
