package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
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
	dialCtx     func(ctx context.Context, network, addr string) (net.Conn, error)

	cookies []*http.Cookie
}

// NewWebClient builds a new instance of WebClient which will provides functions for Http-Ping
func NewWebClient(config *Config) (*WebClient, error) {

	webClient := WebClient{config: config, connCounter: NewConnCounter()}
	webClient.url, _ = url.Parse(config.Target)
	//webClient.httpClient.
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

	webClient.dialCtx = func(ctx context.Context, network, addr string) (net.Conn, error) {

		conn, err := dialer.DialContext(ctx, network, webClient.connTarget)
		if err != nil {
			return conn, err
		}
		return webClient.connCounter.Bind(conn), nil
	}

	webClient.httpClient = &http.Client{
		Timeout: webClient.config.Wait,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	webClient.httpClient.Transport = &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           webClient.dialCtx,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		DisableKeepAlives:     webClient.config.DisableKeepAlive,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: webClient.config.NoCheckCertificate},
	}

	for _, c := range webClient.config.Cookies {
		webClient.cookies = append(webClient.cookies, &http.Cookie{Name: c.Name, Value: c.Value})
	}

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

	for _, c := range webClient.cookies {
		req.AddCookie(c)
	}

	if len(webClient.config.Parameters) > 0 || webClient.config.ExtraParam {
		q := req.URL.Query()

		if webClient.config.ExtraParam {
			q.Add("extra_parameter_httpping", fmt.Sprintf("%X", time.Now().UnixMicro()))
		}

		for _, c := range webClient.config.Parameters {
			q.Add(c.Name, c.Value)
		}
		req.URL.RawQuery = q.Encode()
	}

	start := time.Now()

	req.Header.Set("User-Agent", webClient.config.UserAgent)

	res, err := webClient.httpClient.Do(req)

	if err != nil {
		return &Answer{
			IsFailure:    true,
			FailureCause: "Request timeout",
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

	webClient.cookies = res.Cookies()

	in, out := webClient.connCounter.DeltaAndReset()

	failed := false
	failureCause := ""

	if res.StatusCode/100 == 5 && !webClient.config.IgnoreServerErrors {
		failed = true
		failureCause = "Server-side error"
	}

	return &Answer{
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
