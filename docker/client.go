package docker

import (
	"io"
	"net/http"
	"net/url"
)

// Socket is the default socket location.
const Socket = "/var/run/docker.sock"

type emptyReader struct{}

func (emptyReader) Read([]byte) (int, error) { return 0, io.EOF }

// Client is a docker daemon client that can be used to issue
// docker commands.
type Client struct {
	client *httpClient
}

// NewUnixClient creates a new unix domain socket client.
// When sock is empty, "/var/run/docker.sock" is used.
func NewUnixClient(sock string) *Client {
	if sock == "" {
		sock = Socket
	}
	return &Client{client: newUnixHTTPClient(sock)}
}

func (c *Client) call(
	p string, q url.Values, req, resp interface{},
) error {
	return c.jsonCall(p, q, req, resp)
}

func (c *Client) jsonCall(
	p string, q url.Values, req, resp interface{},
) error {
	u := apiURLQuery(p, q)
	return c.client.call(u, req, resp)
}

func (c *Client) jsonPost(
	p string, q url.Values, req interface{}, w io.Writer,
) error {
	u := apiURLQuery(p, q)
	return c.client.jsonPost(u, req, w)
}

func (c *Client) jsonGet(p string, q url.Values, resp interface{}) error {
	u := apiURLQuery(p, q)
	return c.client.jsonGet(u, resp)
}

func (c *Client) post(
	p string, q url.Values, r io.Reader, w io.Writer,
) error {
	u := apiURLQuery(p, q)
	if r == nil {
		r = emptyReader{}
	}
	return c.client.post(u, r, w)
}

func (c *Client) del(p string, q url.Values) error {
	return c.client.del(apiURLQuery(p, q))
}

func (c *Client) poke(p string, q url.Values) error {
	return c.client.poke(apiURLQuery(p, q))
}

func (c *Client) put(p string, q url.Values, r io.Reader) error {
	u := apiURLQuery(p, q)
	return c.client.put(u, io.NopCloser(r))
}

func (c *Client) get(p string, q url.Values) (*http.Response, error) {
	return c.client.get(apiURLQuery(p, q))
}

func (c *Client) getInto(p string, q url.Values, w io.Writer) (int64, error) {
	return c.client.getInto(apiURLQuery(p, q), w)
}
