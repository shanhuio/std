package fakeweb

import (
	"context"
	"net"
	"net/http"
)

// DialFunc is the signature of the DialContext function used
// in http transport.
type DialFunc func(ctx context.Context, net, addr string) (net.Conn, error)

// SinkDialFunc returns a dialing function that always dials to the same
// address.
func SinkDialFunc(sinkAddr string) DialFunc {
	d := new(net.Dialer)
	return func(ctx context.Context, net, addr string) (net.Conn, error) {
		return d.DialContext(ctx, "tcp", sinkAddr)
	}
}

func sinkClient(sinkAddr string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{DialContext: SinkDialFunc(sinkAddr)},
	}
}

type fakeWebDialer struct {
	dialer    *net.Dialer
	httpAddr  string
	httpsAddr string
}

func newFakeWebDialer(httpAddr, httpsAddr string) *fakeWebDialer {
	return &fakeWebDialer{
		dialer:    new(net.Dialer),
		httpAddr:  httpAddr,
		httpsAddr: httpsAddr,
	}
}

func (d *fakeWebDialer) dial(ctx context.Context, netStr, addr string) (
	net.Conn, error,
) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	sinkAddr := d.httpAddr
	if port == "443" || port == "https" {
		sinkAddr = d.httpsAddr
	}
	return d.dialer.DialContext(ctx, netStr, sinkAddr)
}

func sinkWebDialFunc(httpAddr, httpsAddr string) DialFunc {
	d := newFakeWebDialer(httpAddr, httpsAddr)
	return d.dial
}
