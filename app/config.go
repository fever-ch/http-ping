package app

import (
	"time"
)

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
}
