package cooper

import (
	"io"
	"net"
)

// prefixConn wraps a net.Conn and replaces its Read with a reader that drains
// a byte prefix (leftover HTTP buffer data) before falling through to the
// underlying connection. All other net.Conn methods delegate directly.
type prefixConn struct {
	io.Reader
	net.Conn
}

func (c *prefixConn) Read(b []byte) (int, error) {
	return c.Reader.Read(b)
}
