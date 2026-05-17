package docker

import (
	"context"
	"net"
	"net/http"
	"net/url"
)

func unixSockSink(sinkAddr string) func(
	ctx context.Context, net, addr string,
) (net.Conn, error) {
	d := new(net.Dialer)
	return func(ctx context.Context, net, addr string) (net.Conn, error) {
		return d.DialContext(ctx, "unix", sinkAddr)
	}
}

func unixSockTransport(sockAddr string) *http.Transport {
	return &http.Transport{
		DialContext: unixSockSink(sockAddr),
	}
}

// newUnixHTTPClient creates a new client that always goes to a particular
// unix domain socket.
func newUnixHTTPClient(sockAddr string) *httpClient {
	return &httpClient{
		Server:    &url.URL{Scheme: "http", Host: "unix.sock"},
		Transport: unixSockTransport(sockAddr),
	}
}
