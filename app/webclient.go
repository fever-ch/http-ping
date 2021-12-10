package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}

func (webClient *WebClient) resolve(host string) (*net.IPAddr, error) {
	return net.ResolveIPAddr(webClient.config.IPProtocol, host)
}

// WebClient represents an HTTP/S client designed to do performance analysis
type WebClient struct {
	connCounter *ConnCounter
	httpClient  *http.Client
	reused      bool
	connTarget  string
	config      *Config
	url         *url.URL
}

// NewWebClient builds a new instance of WebClient which will provides functions for Http-Ping
func NewWebClient(config *Config) (*WebClient, error) {

	webClient := WebClient{config: config, connCounter: NewConnCounter()}
	webClient.url, _ = url.Parse(config.Target)

	if config.ConnTarget == "" {

		ipAddr, err := webClient.resolve(webClient.url.Hostname())

		if err != nil {
			return nil, err
		}

		var port = webClient.url.Port()
		if port == "" {
			port = portMap[webClient.url.Scheme]
		}
		webClient.connTarget = fmt.Sprintf("[%s]:%s", ipAddr.IP.String(), port)
	} else {
		webClient.connTarget = config.ConnTarget
	}

	dialer := &net.Dialer{}

	dialCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, webClient.connTarget)
		if err != nil {
			return conn, err
		}
		return webClient.connCounter.Bind(conn), nil
	}

	jar, _ := cookiejar.New(nil)

	webClient.httpClient = &http.Client{
		Jar:     jar,
		Timeout: webClient.config.Wait,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	webClient.httpClient.Transport = &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: dialCtx,

		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.NoCheckCertificate,
		},
		DisableCompression: config.DisableCompression,
		ForceAttemptHTTP2:  true,
		MaxIdleConns:       10,
		DisableKeepAlives:  config.DisableKeepAlive,
		IdleConnTimeout:    config.Interval + config.Wait,
	}

	var cookies []*http.Cookie
	for _, c := range webClient.config.Cookies {
		cookies = append(cookies, &http.Cookie{Name: c.Name, Value: c.Value})
	}

	jar.SetCookies(webClient.url, cookies)

	return &webClient, nil
}

// DoMeasure evaluates the latency to a specific HTTP/S server
func (webClient *WebClient) DoMeasure() *Answer {
	req, _ := http.NewRequest(webClient.config.Method, webClient.config.Target, nil)

	clientTrace := &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			webClient.reused = info.Reused
		},
	}
	traceCtx := httptrace.WithClientTrace(context.Background(), clientTrace)

	req = req.WithContext(traceCtx)

	if len(webClient.config.Parameters) > 0 || webClient.config.ExtraParam {
		q := req.URL.Query()

		if webClient.config.ExtraParam {
			q.Add("extra_parameter_http_ping", fmt.Sprintf("%X", time.Now().UnixMicro()))
		}

		for _, c := range webClient.config.Parameters {
			q.Add(c.Name, c.Value)
		}
		req.URL.RawQuery = q.Encode()
	}

	req.Header.Set("User-Agent", webClient.config.UserAgent)
	if webClient.config.Referrer != "" {
		req.Header.Set("Referer", webClient.config.Referrer)
	}

	if webClient.config.AuthUsername != "" || webClient.config.AuthPassword != "" {
		req.SetBasicAuth(webClient.config.AuthUsername, webClient.config.AuthPassword)
	}

	// Host is considered as a special header in net/http, for simplicity we use here a common way to handle both
	for _, header := range webClient.config.Headers {
		if strings.ToLower(header.Name) != "host" {
			req.Header.Set(header.Name, header.Value)
		} else {
			req.Host = header.Value
		}
	}

	start := time.Now()

	res, err := webClient.httpClient.Do(req)

	if err != nil {
		return &Answer{
			IsFailure:    true,
			FailureCause: err.Error(),
		}
	}

	s, err := io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return &Answer{
			IsFailure:    true,
			FailureCause: "I/O error while reading payload",
		}
	}
	_ = res.Body.Close()
	var d = time.Since(start)

	in, out := webClient.connCounter.DeltaAndReset()

	failed := false
	failureCause := ""

	if res.StatusCode/100 == 5 && !webClient.config.IgnoreServerErrors {
		failed = true
		failureCause = "Server-side error"
	}

	return &Answer{
		Proto:        res.Proto,
		Duration:     d,
		StatusCode:   res.StatusCode,
		Bytes:        s,
		InBytes:      in,
		OutBytes:     out,
		SocketReused: webClient.reused,

		IsFailure:    failed,
		FailureCause: failureCause,
	}

}
