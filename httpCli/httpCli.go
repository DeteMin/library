package httpCli

import (
	"net"
	"net/http"
	"time"
)

// Client the http client wrap
type Client struct {
	*http.Client
}

type options struct {
	// Client
	jar     http.CookieJar
	timeout time.Duration

	// Transport
	keepAlive           time.Duration // default 30
	maxIdleConnsPerHost int           // default 2
	transport           *http.Transport
}

// Option the params of http which can self-defined
type Option func(*options)

// WithTimeout set the timeout of request
func WithTimeout(t time.Duration) Option {
	return func(o *options) {
		o.timeout = t
	}
}

// WithCookieJar set the CookieJar of request
func WithCookieJar(cj http.CookieJar) Option {
	return func(o *options) {
		o.jar = cj
	}
}

// WithTransport set the Transport of your own
func WithTransport(ts *http.Transport) Option {
	return func(o *options) {
		o.transport = ts
	}
}

// MaxIdleConnsPerHost set the max idle connects per host
func MaxIdleConnsPerHost(n int) Option {
	return func(o *options) {
		o.maxIdleConnsPerHost = n
	}
}

// KeepAlive set the connection keep live time
func KeepAlive(t time.Duration) Option {
	return func(o *options) {
		o.keepAlive = t
	}
}

// NewClient return a http client wrap to deal with http request
func NewClient(opt ...Option) *Client {
	opts := options{
		timeout:             5 * time.Second, // 请求超时时间
		keepAlive:           30 * time.Second,
		maxIdleConnsPerHost: 2, // 请求量较大时需调整此参数,否则会出现fd被耗尽,出现大量TIME_WAIT
	}
	for _, o := range opt {
		o(&opts)
	}
	var ts *http.Transport
	if opts.transport != nil {
		ts = opts.transport
	} else {
		ts = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: opts.keepAlive,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   opts.maxIdleConnsPerHost,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
	}

	return &Client{
		&http.Client{
			Timeout:   opts.timeout,
			Jar:       opts.jar,
			Transport: ts,
		}}
}

// Head create a new http head request
func (c *Client) Head(uri string) *Request {
	return newRequest(c, http.MethodHead, uri)
}

// Get create a new http get request
func (c *Client) Get(uri string) *Request {
	return newRequest(c, http.MethodGet, uri)
}

// Post create a new http post request
func (c *Client) Post(uri string) *Request {
	return newRequest(c, http.MethodPost, uri)
}

// Put create a new http put request
func (c *Client) Put(uri string) *Request {
	return newRequest(c, http.MethodPut, uri)
}

// Delete create a new http delete request
func (c *Client) Delete(uri string) *Request {
	return newRequest(c, http.MethodDelete, uri)
}
