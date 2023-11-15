package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

type HttpClient struct {
	http.Client
	transport *http.Transport
	UserAgent string
}

var (
	SimpleHttpClient = NewHttpClient()
)

func NewHttpClient() *HttpClient {
	hc := HttpClient{
		Client: http.Client{
			Timeout: 400 * time.Second,
			/*
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					return http.ErrUseLastResponse
				},
			*/
		},
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; WOW64; rv:88.0) Gecko/20100101 Firefox/88.0",
	}
	hc.Client.Jar, _ = cookiejar.New(nil)
	hc.lazyInit()
	return &hc
}
func (h *HttpClient) lazyInit() {
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

//such as http://127.0.0.1:8080
// socks5://127.0.0.1:1080
func (hc *HttpClient) SetProxy(proxy string) error {
	if proxy == "" {
		hc.transport.Proxy = nil
		return nil
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return err
	}
	hc.transport.Proxy = http.ProxyURL(u)
	return nil
}

func HandleRespon2String(resp *http.Response, err error) (string, error) {
	if resp == nil {
		return "", err
	}
	defer resp.Body.Close()
	if err != nil {
		return "", err
	}
	buff, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buff), nil
}
func (hc *HttpClient) setHttpBaseHeader(req *http.Request, header map[string]string) {
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
func (hc *HttpClient) setHttpContentType(req *http.Request, method string) {
	//last check header
	if method == "POST" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}
func (hc *HttpClient) HttpSimpleFormPost(URL string, postBody map[string]string) (*http.Response, error) {
	return hc.HttpFormPost(URL, nil, postBody)
}
func (hc *HttpClient) HttpFormPost(URL string, header map[string]string, postBody map[string]string) (*http.Response, error) {
	tpostBody := map[string][]string{}
	if postBody != nil {
		for k, v := range postBody {
			tpostBody[k] = []string{v}
		}
	}
	var body url.Values = tpostBody
	return hc.HttpPost(URL, header, body.Encode())
}
func (hc *HttpClient) HttpPost(URL string, header map[string]string, postBody string) (*http.Response, error) {
	return hc.HttpRequest("POST", URL, header, postBody)
}
func (hc *HttpClient) HttpSimpleGet(URL string) (*http.Response, error) {
	return hc.HttpGet(URL, nil, nil)
}
func (hc *HttpClient) HttpGet(URL string, header map[string]string, params map[string]string) (*http.Response, error) {
	paramsVal := url.Values{}
	realURL := URL
	if params != nil {
		for k, v := range params {
			paramsVal.Add(k, v)
		}
		realURL = URL + "?" + paramsVal.Encode()
	}
	return hc.HttpRequest("GET", realURL, header, "")
}
func (hc *HttpClient) HttpRequest(method string, URL string, header map[string]string, postBody string) (*http.Response, error) {
	md := strings.ToUpper(method)
	req, err := http.NewRequest(md, URL, strings.NewReader(postBody))
	if err != nil {
		return nil, err
	}
	hc.setHttpBaseHeader(req, header)
	hc.setHttpContentType(req, md)
	resp, err := hc.Do(req)
	return resp, err
}

func GetDownloadFileName(u *url.URL, fileName string, disposition string) string {
	if disposition != "" && strings.Contains(disposition, "filename") {
		_, params, err := mime.ParseMediaType(disposition)
		if err == nil {
			return params["filename"]
		}
	}
	fn := strings.TrimSpace(fileName)
	if fn != "" {
		return fn
	}
	idx := strings.LastIndex(u.Path, "/")
	if idx > -1 {
		if u.Path == "/" {
			return "index.html"
		}
		return u.Path[idx+1:]
	}
	return u.Path
}

func (hc *HttpClient) HttpGetDownloadFile(URL string, fileName string) error {
	resp, err := hc.HttpGet(URL, nil, nil)
	if err != nil {
		return errors.New(fmt.Sprint("download ", URL, " failed", err))
	}
	defer resp.Body.Close()
	//contentType := resp.Header.Get("Content-Type")
	sc := resp.StatusCode / 100
	if sc != 2 {
		return fmt.Errorf("request errors,status code = %d  status =  %s", sc, resp.Status)
	}

	realName := GetDownloadFileName(resp.Request.URL, fileName, resp.Header.Get("Content-Disposition"))
	dst, err := os.Create(realName)
	if err != nil {
		return errors.New(fmt.Sprint("create file ", fileName, " failed ", err))
	}
	defer dst.Close()
	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		return fmt.Errorf("write file %s to failed err = %s", fileName, err.Error())
	}
	return nil
}
