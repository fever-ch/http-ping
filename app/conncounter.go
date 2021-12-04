package app

import (
	"net"
	"sync/atomic"
	"time"
)

// ConnCounter contains the counters to all the connections which might be bound to it (bind)
type ConnCounter struct {
	in  int64
	out int64
}

// NewConnCounter builds a new instance of ConnCounter which collects some stats on network connections
func NewConnCounter() *ConnCounter {
	return &ConnCounter{0, 0}
}

// Bind function builds a proxy which will count all the I/O done to a connection
func (cc *ConnCounter) Bind(conn net.Conn) net.Conn {
	return &connCounterInstance{counter: cc, innerConn: conn}
}

type connCounterInstance struct {
	counter   *ConnCounter
	innerConn net.Conn
}

// Read behaves is a proxy to the actual conn.Read (counts reads)
func (c *connCounterInstance) Read(b []byte) (int, error) {
	n, err := c.innerConn.Read(b)
	if err == nil {
		atomic.AddInt64(&c.counter.in, int64(n))
	}
	return n, err
}

// Write behaves is a proxy to the actual conn.Write (counts writes)
func (c *connCounterInstance) Write(b []byte) (int, error) {
	n, err := c.innerConn.Write(b)
	if err == nil {
		atomic.AddInt64(&c.counter.out, int64(n))
	}
	return n, err
}

// Close behaves is a proxy to the actual conn.Close
func (c *connCounterInstance) Close() error {
	return c.innerConn.Close()
}

// LocalAddr behaves is a proxy to the actual conn.LocalAddr
func (c *connCounterInstance) LocalAddr() net.Addr {
	return c.innerConn.LocalAddr()
}

// RemoteAddr behaves is a proxy to the actual conn.RemoteAddr
func (c *connCounterInstance) RemoteAddr() net.Addr {
	return c.innerConn.LocalAddr()
}

// SetDeadline behaves is a proxy to the actual conn.SetDeadline
func (c *connCounterInstance) SetDeadline(t time.Time) error {
	return c.innerConn.SetDeadline(t)
}

// SetReadDeadline behaves is a proxy to the actual conn.SetReadDeadline
func (c *connCounterInstance) SetReadDeadline(t time.Time) error {
	return c.innerConn.SetReadDeadline(t)
}

// SetWriteDeadline behaves is a proxy to the actual conn.SetWriteDeadline
func (c *connCounterInstance) SetWriteDeadline(t time.Time) error {
	return c.innerConn.SetWriteDeadline(t)
}

// DeltaAndReset retrieve the counters and reset them atomically
func (cc *ConnCounter) DeltaAndReset() (int64, int64) {
	return atomic.SwapInt64(&cc.in, 0), atomic.SwapInt64(&cc.out, 0)
}
