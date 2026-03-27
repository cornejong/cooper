package cooper

import (
	"io"
	"net"
	"strings"
	"testing"
)

func TestPrefixConn_ReadDrainsPrefixFirst(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	prefix := "leftover"
	pc := &prefixConn{
		Reader: io.MultiReader(strings.NewReader(prefix), client),
		Conn:   client,
	}

	go func() {
		server.Write([]byte(" from conn"))
		server.Close()
	}()

	all, err := io.ReadAll(pc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	expected := "leftover from conn"
	if string(all) != expected {
		t.Fatalf("expected %q, got %q", expected, string(all))
	}
}

func TestPrefixConn_EmptyPrefix(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	pc := &prefixConn{
		Reader: io.MultiReader(strings.NewReader(""), client),
		Conn:   client,
	}

	go func() {
		server.Write([]byte("just conn"))
		server.Close()
	}()

	all, err := io.ReadAll(pc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(all) != "just conn" {
		t.Fatalf("expected %q, got %q", "just conn", string(all))
	}
}

func TestPrefixConn_WriteDelegatesToUnderlying(t *testing.T) {
	server, client := net.Pipe()

	pc := &prefixConn{
		Reader: strings.NewReader(""),
		Conn:   client,
	}

	done := make(chan string)
	go func() {
		buf := make([]byte, 128)
		n, _ := server.Read(buf)
		done <- string(buf[:n])
		server.Close()
	}()

	msg := "written through prefixConn"
	if _, err := pc.Write([]byte(msg)); err != nil {
		t.Fatalf("write: %v", err)
	}

	got := <-done
	if got != msg {
		t.Fatalf("expected %q, got %q", msg, got)
	}

	pc.Close()
}

func TestPrefixConn_SmallReadBuffer(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	prefix := "abcdefghij"
	pc := &prefixConn{
		Reader: io.MultiReader(strings.NewReader(prefix), client),
		Conn:   client,
	}

	go func() {
		server.Write([]byte("klmnop"))
		server.Close()
	}()

	// read in small 3-byte chunks
	var result []byte
	buf := make([]byte, 3)
	for {
		n, err := pc.Read(buf)
		result = append(result, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read: %v", err)
		}
	}

	expected := "abcdefghijklmnop"
	if string(result) != expected {
		t.Fatalf("expected %q, got %q", expected, string(result))
	}
}
