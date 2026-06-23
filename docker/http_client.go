package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"

	"shanhu.io/std/errcode"
)

// httpClient performs calls to a remote server.
type httpClient struct {
	Server    *url.URL
	Transport http.RoundTripper
}

func makeURL(base *url.URL, p string) (string, error) {
	u := *base
	up, err := url.Parse(p)
	if err != nil {
		return "", err
	}

	// append two paths
	u.Path = path.Join(u.Path, up.Path)
	u.RawQuery = up.RawQuery
	u.Fragment = up.Fragment
	return u.String(), nil
}

func copyRespBody(resp *http.Response, w io.Writer) error {
	defer resp.Body.Close()
	if w == nil {
		return nil
	}
	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *httpClient) doRaw(ctx context.Context, req *http.Request) (
	*http.Response, error,
) {
	return (&http.Client{Transport: c.Transport}).Do(req.WithContext(ctx))
}

func (c *httpClient) do(ctx context.Context, req *http.Request) (
	*http.Response, error,
) {
	resp, err := c.doRaw(ctx, req)
	if err != nil {
		return nil, err
	}
	if !isSuccess(resp) {
		defer resp.Body.Close()
		return nil, httpRespError(resp)
	}
	return resp, nil
}

func (c *httpClient) req(m, p string, r io.Reader) (*http.Request, error) {
	u, err := makeURL(c.Server, p)
	if err != nil {
		return nil, err
	}
	return http.NewRequest(m, u, r)
}

func (c *httpClient) reqJSON(m, p string, r io.Reader) (*http.Request, error) {
	req, err := c.req(m, p, r)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *httpClient) put(p string, r io.Reader) error {
	req, err := c.req(http.MethodPut, p, r)
	if err != nil {
		return err
	}
	resp, err := c.do(context.TODO(), req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *httpClient) pokeMethod(ctx context.Context, m, p string) error {
	req, err := c.req(m, p, nil)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func (c *httpClient) poke(p string) error {
	return c.pokeMethod(context.TODO(), http.MethodPost, p)
}

func (c *httpClient) get(p string) (*http.Response, error) {
	req, err := c.req(http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	return c.do(context.TODO(), req)
}

func (c *httpClient) getInto(p string, w io.Writer) (int64, error) {
	resp, err := c.get(p)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return io.Copy(w, resp.Body)
}

func (c *httpClient) jsonGet(p string, resp any) error {
	req, err := c.reqJSON(http.MethodGet, p, nil)
	if err != nil {
		return nil
	}
	httpResp, err := c.do(context.TODO(), req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	dec := json.NewDecoder(httpResp.Body)
	if err := dec.Decode(resp); err != nil {
		return err
	}
	return httpResp.Body.Close()
}

func (c *httpClient) post(p string, r io.Reader, w io.Writer) error {
	if r != nil {
		r = io.NopCloser(r)
	}
	req, err := c.req(http.MethodPost, p, r)
	if err != nil {
		return err
	}
	resp, err := c.do(context.TODO(), req)
	if err != nil {
		return err
	}
	return copyRespBody(resp, w)
}

func (c *httpClient) postJSON(ctx context.Context, p string, req any) (
	*http.Response, error,
) {
	bs, err := json.Marshal(req)
	if err != nil {
		return nil, errcode.Annotate(err, "marshal request")
	}
	httpReq, err := c.reqJSON(http.MethodPost, p, bytes.NewBuffer(bs))
	if err != nil {
		return nil, err
	}
	return c.do(ctx, httpReq)
}

func (c *httpClient) jsonPost(p string, req any, w io.Writer) error {
	resp, err := c.postJSON(context.TODO(), p, req)
	if err != nil {
		return err
	}
	return copyRespBody(resp, w)
}

func (c *httpClient) call(p string, req, resp any) error {
	httpResp, err := c.postJSON(context.TODO(), p, req)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if resp == nil {
		return nil
	}
	dec := json.NewDecoder(httpResp.Body)
	if err := dec.Decode(resp); err != nil {
		return errcode.Annotate(err, "decode response")
	}
	return httpResp.Body.Close()
}

func (c *httpClient) del(p string) error {
	return c.pokeMethod(context.TODO(), http.MethodDelete, p)
}
