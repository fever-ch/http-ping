package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/fever-ch/http-ping/net/sockettrace"
	"github.com/fever-ch/http-ping/stats"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}

// WebClient represents an HTTP/S client designed to do performance analysis
type WebClient struct {
	httpClient *http.Client
	connTarget string
	config     *Config
	url        *url.URL
	resolver   *resolver

	writes int64
	reads  int64
}

func init() {
	// load system cert pool once at the beginning to not impact further measures
	_, _ = x509.SystemCertPool()
}

// NewWebClient builds a new instance of WebClient which will provides functions for Http-Ping
func NewWebClient(config *Config) (*WebClient, error) {
	webClient := WebClient{config: config}
	webClient.url, _ = url.Parse(config.Target)
	webClient.resolver = newResolver(config)

	if config.ConnTarget == "" {
		webClient.connTarget = webClient.url.Hostname()
		ipAddr := webClient.url.Hostname()

		var port = webClient.url.Port()
		if port == "" {
			port = portMap[webClient.url.Scheme]
		}

		if strings.Contains(ipAddr, ":") {
			webClient.connTarget = fmt.Sprintf("[%s]:%s", ipAddr, port)
		} else {
			webClient.connTarget = fmt.Sprintf("%s:%s", ipAddr, port)
		}
	} else {
		webClient.connTarget = config.ConnTarget
	}

	dialer := &net.Dialer{}

	dialer.Resolver = &net.Resolver{
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			addr, _ := webClient.resolver.resolveConn(address)
			return net.Dial(network, addr)
		}}

	startDNSHook := func(ctx context.Context) {
		trace := httptrace.ContextClientTrace(ctx)
		if trace == nil || trace.DNSStart == nil {
			return
		}

		trace.DNSStart(httptrace.DNSStartInfo{})
	}

	stopDNSHook := func(ctx context.Context) {
		trace := httptrace.ContextClientTrace(ctx)
		if trace == nil || trace.DNSDone == nil {
			return
		}

		trace.DNSDone(httptrace.DNSDoneInfo{})
	}

	dialCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {

		startDNSHook(ctx)
		ipaddr, _ := webClient.resolver.resolveConn(webClient.connTarget)
		stopDNSHook(ctx)

		return sockettrace.NewSocketTrace(ctx, dialer, network, ipaddr)
	}

	webClient.httpClient = &http.Client{
		Timeout: webClient.config.Wait,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy:       http.ProxyFromEnvironment,
			DialContext: dialCtx,

			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.NoCheckCertificate,
			},
			DisableCompression: config.DisableCompression,
			ForceAttemptHTTP2:  !webClient.config.DisableHTTP2,
			MaxIdleConns:       10,
			DisableKeepAlives:  config.DisableKeepAlive,
			IdleConnTimeout:    config.Interval + config.Wait,
		},
	}

	if webClient.config.DisableHTTP2 {
		webClient.httpClient.Transport.(*http.Transport).TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	}

	return &webClient, nil
}

// DoMeasure evaluates the latency to a specific HTTP/S server
func (webClient *WebClient) DoMeasure() *HTTPMeasure {
	req, _ := http.NewRequest(webClient.config.Method, webClient.config.Target, nil)

	if webClient.httpClient.Jar == nil || !webClient.config.KeepCookies {
		jar, _ := cookiejar.New(nil)
		var cookies []*http.Cookie
		for _, c := range webClient.config.Cookies {
			cookies = append(cookies, &http.Cookie{Name: c.Name, Value: c.Value})
		}

		jar.SetCookies(webClient.url, cookies)
		webClient.httpClient.Jar = jar
	}

	var reused bool
	var remoteAddr string

	connTimer := newTimer()
	dnsTimer := newTimer()
	tlsTimer := newTimer()
	tcpTimer := newTimer()
	reqTimer := newTimer()
	waitTimer := newTimer()
	responseTimer := newTimer()

	clientTrace := &httptrace.ClientTrace{

		TLSHandshakeStart: func() {
			tlsTimer.start()
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsTimer.stop()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsTimer.start()
		},

		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsTimer.stop()
		},

		GetConn: func(hostPort string) {
			connTimer.start()
		},

		GotConn: func(info httptrace.GotConnInfo) {
			remoteAddr = info.Conn.RemoteAddr().String()
			connTimer.stop()
			reqTimer.start()
			reused = info.Reused
		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {
			reqTimer.stop()
			waitTimer.start()

		},

		GotFirstResponseByte: func() {
			waitTimer.stop()
			responseTimer.start()
		},
	}

	ctx := sockettrace.WithTrace(context.Background(),
		&sockettrace.ConnTrace{
			Read: func(i int) {
				atomic.AddInt64(&webClient.reads, int64(i))
			},
			Write: func(i int) {
				atomic.AddInt64(&webClient.writes, int64(i))
			},
			TCPStart: func() {
				tcpTimer.start()
			},
			TCPEstablished: func() {
				tcpTimer.stop()
			},
		})

	traceCtx := httptrace.WithClientTrace(ctx, clientTrace)

	req = req.WithContext(traceCtx)

	if len(webClient.config.Parameters) > 0 || webClient.config.ExtraParam {
		q := req.URL.Query()

		if webClient.config.ExtraParam {
			q.Add("extra_parameter_http_ping", fmt.Sprintf("%d", time.Now().UnixMicro()))
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
	//reqTimer.start()

	res, err := webClient.httpClient.Do(req)

	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: err.Error(),
		}
	}

	s, err := io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return &HTTPMeasure{
			IsFailure:    true,
			FailureCause: "I/O error while reading payload",
		}
	}

	_ = res.Body.Close()
	responseTimer.stop()
	var d = time.Since(start)

	failed := false
	failureCause := ""

	if res.StatusCode/100 == 5 && !webClient.config.IgnoreServerErrors {
		failed = true
		failureCause = "Server-side error"
	}

	//fmt.Printf("dns:  %d\n", dnsTimer.duration().Milliseconds())
	//fmt.Printf("tcp:  %d\n", tcpTimer.duration().Milliseconds())
	//fmt.Printf("tls:  %d\n", tlsTimer.duration().Milliseconds())
	//fmt.Printf("conn: %d\n", connTimer.duration().Milliseconds())
	//fmt.Printf("req:  %d\n", reqTimer.duration().Milliseconds())

	i := atomic.SwapInt64(&webClient.reads, 0)
	o := atomic.SwapInt64(&webClient.writes, 0)

	var tlsVersion string
	if res.TLS != nil {
		code := int(res.TLS.Version) - 0x0301
		if code >= 0 {
			tlsVersion = fmt.Sprintf("TLS-1.%d", code)
		} else {
			tlsVersion = "SSL-3"
		}
	}

	return &HTTPMeasure{
		Proto:        res.Proto,
		Measure:      stats.Measure(d),
		StatusCode:   res.StatusCode,
		Bytes:        s,
		InBytes:      i,
		OutBytes:     o,
		SocketReused: reused,
		Compressed:   !res.Uncompressed,
		TLSEnabled:   res.TLS != nil,
		TLSVersion:   tlsVersion,

		DNSDuration:  dnsTimer.measure(),
		TCPHandshake: tcpTimer.measure(),
		TLSDuration:  tlsTimer.measure(),
		ConnDuration: connTimer.measure(),
		ReqDuration:  reqTimer.measure(),
		Wait:         waitTimer.measure(),
		RespDuration: responseTimer.measure(),

		RemoteAddr: remoteAddr,

		IsFailure:    failed,
		FailureCause: failureCause,
	}

}
