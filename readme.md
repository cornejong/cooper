# cooper

Simple HTTP hijacking for Go.

Does the `101 Switching Protocols` handshake, deals with buffered bytes, and gives you a `net.Conn`. That's it.

```sh
go get github.com/cornejong/cooper
```

## Server side

`cooper.Hijack` gives you an `http.Handler`. It does the upgrade handshake and hands you the raw connection:

```go
handler := cooper.Hijack(func(conn net.Conn, proto string) {
    defer conn.Close()
    fmt.Fprintf(conn, "hello over %s!\n", proto)
}, "myproto/1")

http.Handle("/connect", handler)
http.ListenAndServe(":8080", nil)
```

Pass one or more protocol names to restrict what the server accepts. If none are passed, any `Upgrade` value goes through. Protocol matching is case-insensitive. Clients requesting something not in the list get a `426 Upgrade Required`.

The handler receives the connection and is responsible for closing it.

## Client side

`cooper.Upgrade` does the client half over an existing `net.Conn`:

```go
conn, _ := net.Dial("tcp", "localhost:8080")

req, _ := http.NewRequest("GET", "http://localhost:8080/connect", nil)
req.Header.Set("Upgrade", "myproto/1")

upgraded, err := cooper.Upgrade(conn, req)
if err != nil {
    panic(err)
}
defer upgraded.Close()

buf := make([]byte, 1024)
n, _ := upgraded.Read(buf)
fmt.Println(string(buf[:n]))
```

You build the `*http.Request` yourself. The only requirement is that it has an `Upgrade` header set. `Connection: Upgrade` gets added if you forget it.

There's a 10 second timeout on the handshake. If the server doesn't respond with `101`, or the `Upgrade`/`Connection` headers in the response don't check out, you get an error back.

## Buffered bytes

Go's HTTP server (and `bufio.Reader` on the client side) might read past the end of the handshake into your protocol data. Cooper handles this internally with a `prefixConn` wrapper that drains those buffered bytes first before reading from the underlying connection. You don't need to think about it.
