package http

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// HTTPClient wraps http.Client with a shared User-Agent and a lazily built
// transport tuned for long-running transfers.
type HTTPClient struct {
	http.Client
	transport *http.Transport
	UserAgent string
}

// SimpleHTTPClient is a shared default client.
var SimpleHTTPClient = NewHTTPClient()

// NewHTTPClient creates an HTTPClient with sensible timeouts and a cookie jar.
func NewHTTPClient() *HTTPClient {
	hc := HTTPClient{
		Client: http.Client{
			Timeout: 400 * time.Second,
		},
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; WOW64; rv:88.0) Gecko/20100101 Firefox/88.0",
	}
	hc.Client.Jar, _ = cookiejar.New(nil)
	hc.lazyInit()
	return &hc
}

func (h *HTTPClient) lazyInit() {
	if h.transport == nil {
		h.transport = &http.Transport{
			Proxy: nil,
			DialContext: (&net.Dialer{
				Timeout:   40 * time.Second,
				KeepAlive: 40 * time.Second,
				DualStack: true,
			}).DialContext,
			TLSHandshakeTimeout:   90 * time.Second,
			DisableKeepAlives:     false,
			DisableCompression:    false, // gzip
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 30 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}
		h.Client.Transport = h.transport
	}
}

// ResponseToString reads the whole response body into a string.
func ResponseToString(resp *http.Response, err error) (string, error) {
	if resp == nil {
		return "", err
	}
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	buff, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buff), nil
}

func (hc *HTTPClient) setHTTPBaseHeader(req *http.Request, header map[string]string) {
	if header != nil {
		for k, v := range header {
			req.Header.Add(k, v)
		}
		//change Host header
		if header["Host"] != "" {
			v := header["Host"]
			if req.URL.Port() == "80" {
				req.Host = v
			} else {
				req.Host = v + ":" + req.URL.Port()
			}
		}
	}
	//last check header
	if req.Header.Get("Accept") == "" {
		req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}
	if hc.UserAgent != "" {
		req.Header.Add("User-Agent", hc.UserAgent)
	}
}

func (hc *HTTPClient) setHTTPContentType(req *http.Request, method string) {
	//last check header
	if method == "POST" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

// HTTPFormPost posts form-encoded fields.
func (hc *HTTPClient) HTTPFormPost(URL string, header map[string]string, postBody map[string]string) (*http.Response, error) {
	tpostBody := map[string][]string{}
	if postBody != nil {
		for k, v := range postBody {
			tpostBody[k] = []string{v}
		}
	}
	var body url.Values = tpostBody
	return hc.HTTPPost(URL, header, body.Encode())
}

// HTTPPost issues an HTTP POST with the given string body.
func (hc *HTTPClient) HTTPPost(URL string, header map[string]string, postBody string) (*http.Response, error) {
	return hc.HTTPRequest("POST", URL, header, postBody)
}

// HTTPGet issues an HTTP GET, appending params to the query string when given.
func (hc *HTTPClient) HTTPGet(URL string, header map[string]string, params map[string]string) (*http.Response, error) {
	paramsVal := url.Values{}
	realURL := URL
	if params != nil {
		for k, v := range params {
			paramsVal.Add(k, v)
		}
		realURL = URL + "?" + paramsVal.Encode()
	}
	return hc.HTTPRequest("GET", realURL, header, "")
}

// HTTPRequest issues an arbitrary HTTP method.
func (hc *HTTPClient) HTTPRequest(method string, URL string, header map[string]string, postBody string) (*http.Response, error) {
	md := strings.ToUpper(method)
	req, err := http.NewRequest(md, URL, strings.NewReader(postBody))
	if err != nil {
		return nil, err
	}
	hc.setHTTPBaseHeader(req, header)
	hc.setHTTPContentType(req, md)
	resp, err := hc.Do(req)
	return resp, err
}
