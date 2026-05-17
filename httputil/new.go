package httputil

import (
	"net/url"
)

// NewUnixClient creates a new client that always goes to a particular
// unix domain socket.
func NewUnixClient(sockAddr string) *Client {
	return &Client{
		Server:    &url.URL{Scheme: "http", Host: "unix.sock"},
		Transport: unixSockTransport(sockAddr),
	}
}
