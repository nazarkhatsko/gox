// Package httpx is an HTTP client for calling APIs: one shared connection
// pool, a timeout for requests whose context has no deadline, JSON request
// bodies, and responses read whole. It knows nothing of who uses it: plain
// values in, plain values out.
//
//	c := httpx.New(httpx.Options{Header: http.Header{"Authorization": {"Bearer " + token}}})
//
//	resp, err := c.Do(ctx, httpx.Request{Method: http.MethodPost, URL: url, JSON: payload})
//	if err != nil {
//		return err
//	}
//
//	var out Result
//	err = resp.DecodeJSON(&out)
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

// DefaultTimeout bounds a request whose context has no deadline, when
// Options.Timeout is 0. Without it a hung server would hold the caller
// forever.
const DefaultTimeout = 5 * time.Minute

// ErrRequest is wrapped by Do's errors about the request itself (a JSON
// body that doesn't marshal, a malformed URL or method), as opposed to
// sending it or reading the response.
var ErrRequest = errors.New("invalid request")

// transport is shared by every Client without its own: one connection
// pool, so many clients calling the same host reuse connections instead of
// opening one per request.
var transport = newTransport()

func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 512
	t.MaxIdleConnsPerHost = 64

	return t
}

// Options configures a Client.
type Options struct {
	// Header is sent with every request; a request's own header is set
	// over it, key by key.
	Header http.Header
	// Timeout bounds a request whose context has no deadline, reading the
	// response included: 0 is DefaultTimeout, a negative one is none.
	Timeout time.Duration
	// Transport sends the requests; nil is the shared pool.
	Transport http.RoundTripper
}

// Client sends requests. It is safe for concurrent use.
type Client struct {
	header  http.Header
	timeout time.Duration
	http    *http.Client
}

// New returns a client configured by opts.
func New(opts Options) *Client {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	var rt http.RoundTripper = transport
	if opts.Transport != nil {
		rt = opts.Transport
	}

	return &Client{header: canonical(opts.Header), timeout: timeout, http: &http.Client{Transport: rt}}
}

// Request is a request to send.
type Request struct {
	Method string
	URL    string
	// Header is set over the client's, key by key.
	Header http.Header
	// Body is sent as is. Ignored if JSON is set.
	Body []byte
	// JSON, if not nil, is sent marshaled, with Content-Type
	// application/json (unless Header sets another).
	JSON any
}

// Response is what a request got. A non-2xx status is a response, not an
// error.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	// RequestHeader is the header sent: the client's, the request's and
	// what Do added (Content-Type).
	RequestHeader http.Header
}

// DecodeJSON unmarshals the body into v.
func (r Response) DecodeJSON(v any) error {
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// Do sends r and reads the whole response. Errors about r itself wrap
// ErrRequest; the others are about sending it or reading the response.
func (c *Client) Do(ctx context.Context, r Request) (Response, error) {
	body := r.Body

	if r.JSON != nil {
		b, err := json.Marshal(r.JSON)
		if err != nil {
			return Response{}, fmt.Errorf("%w: marshal body: %w", ErrRequest, err)
		}

		body = b
	}

	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, reader)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrRequest, err)
	}

	if r.JSON != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for _, h := range []http.Header{c.header, canonical(r.Header)} {
		for k, vs := range h {
			req.Header[k] = slices.Clone(vs)
		}
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("%s %s: %w", req.Method, req.URL.Redacted(), err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}

	return Response{Status: resp.StatusCode, Header: resp.Header, Body: data, RequestHeader: req.Header}, nil
}

// canonical returns h with its keys in canonical form, so a key set over
// another written differently ("x-api-key", "X-Api-Key") replaces it.
func canonical(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, vs := range h {
		key := http.CanonicalHeaderKey(k)
		out[key] = append(out[key], vs...)
	}

	return out
}
