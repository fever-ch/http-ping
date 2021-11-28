package app

import (
	"net"
	"time"
)

type ConnCounter struct {
	in  int64
	out int64
}

// NewConnCounter builds a new instance of ConnCounter which collects some stats on network connections
func NewConnCounter() *ConnCounter {
	return &ConnCounter{0, 0}
}

func (cc *ConnCounter) Bind(conn net.Conn) net.Conn {
	return &ConnCounterInstance{counter: cc, innerConn: conn}
}

type ConnCounterInstance struct {
	counter   *ConnCounter
	innerConn net.Conn
}

func (c *ConnCounterInstance) Read(b []byte) (int, error) {
	n, err := c.innerConn.Read(b)
	if err == nil {
		c.counter.in += int64(n)
	}
	return n, err
}

func (c *ConnCounterInstance) Write(b []byte) (int, error) {
	n, err := c.innerConn.Write(b)
	if err == nil {
		c.counter.out += int64(n)
	}
	return n, err
}

func (c *ConnCounterInstance) Close() error {
	return c.innerConn.Close()
}

func (c *ConnCounterInstance) LocalAddr() net.Addr {
	return c.innerConn.LocalAddr()
}

func (c *ConnCounterInstance) RemoteAddr() net.Addr {
	return c.innerConn.LocalAddr()
}

func (c *ConnCounterInstance) SetDeadline(t time.Time) error {
	return c.innerConn.SetDeadline(t)
}
func (c *ConnCounterInstance) SetReadDeadline(t time.Time) error {
	return c.innerConn.SetReadDeadline(t)
}

func (c *ConnCounterInstance) SetWriteDeadline(t time.Time) error {
	return c.innerConn.SetWriteDeadline(t)
}
