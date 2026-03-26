package cooper

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// handshakeTimeout caps how long the HTTP upgrade round-trip may take.
const handshakeTimeout = 10 * time.Second

// Upgrade performs an HTTP/1.1 protocol upgrade on conn using req. On success
// it returns a net.Conn.
//
// The caller is fully responsible for constructing req: method, URL, headers,
// and an optional body may all be set. The only hard requirement is that
// req.Header must contain "Upgrade: <proto>" (used for response validation)
func Upgrade(conn net.Conn, req *http.Request) (net.Conn, error) {
	proto := req.Header.Get("Upgrade")
	if proto == "" {
		return nil, fmt.Errorf("upgrade: request is missing Upgrade header")
	}

	if !strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade") {
		req.Header.Set("Connection", "upgrade")
	}

	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return nil, fmt.Errorf("upgrade %q: set deadline: %w", proto, err)
	}
	defer conn.SetDeadline(time.Time{})

	if err := req.Write(conn); err != nil {
		return nil, fmt.Errorf("upgrade %q: send request: %w", proto, err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, fmt.Errorf("upgrade %q: read response: %w", proto, err)
	}
	resp.Body.Close()

	// Require 101 Switching Protocols.
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("upgrade %q: unexpected status %s", proto, resp.Status)
	}

	if !strings.EqualFold(resp.Header.Get("Upgrade"), proto) {
		return nil, fmt.Errorf("upgrade %q: server returned different upgrade value %q",
			proto, resp.Header.Get("Upgrade"))
	}

	if !strings.Contains(strings.ToLower(resp.Header.Get("Connection")), "upgrade") {
		return nil, fmt.Errorf("upgrade %q: response missing Connection: Upgrade header", proto)
	}

	if n := br.Buffered(); n > 0 {
		peeked := make([]byte, n)
		if _, err := io.ReadFull(br, peeked); err != nil {
			return nil, fmt.Errorf("upgrade %q: drain read buffer: %w", proto, err)
		}

		return &prefixConn{
			Reader: io.MultiReader(bytes.NewReader(peeked), conn),
			Conn:   conn,
		}, nil
	}

	return conn, nil
}
