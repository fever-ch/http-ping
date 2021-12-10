package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/domainr/dnsr"
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
	if webClient.config.FullDNS {
		var ip net.IP

		if ip = net.ParseIP(host); ip == nil {
			if entries, err := fullResolveFromRoot(webClient.config.IPProtocol, host); err == nil {
				ip = net.ParseIP(*entries)
			}
		}
		if ip != nil {
			return &net.IPAddr{IP: ip}, nil
		}
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}

	}
	return net.ResolveIPAddr(webClient.config.IPProtocol, host)

}

func fullResolveFromRoot(network, host string) (*string, error) {
	var qtypes []string

	if network == "ip" {
		qtypes = []string{"A", "AAAA"}
	} else if network == "ip4" {
		qtypes = []string{"A"}
	} else if network == "ip6" {
		qtypes = []string{"AAAA"}
	} else {
		qtypes = []string{}
	}

	r := dnsr.New(1024)
	requestCount := 0

	var resolveRecu func(r *dnsr.Resolver, host string) (*string, error)

	resolveRecu = func(r *dnsr.Resolver, host string) (*string, error) {
		requestCount++

		cnames := make(map[string]struct{})
		for _, qtype := range qtypes {
			for _, rr := range r.Resolve(host, qtype) {
				if rr.Type == qtype {
					return &rr.Value, nil
				} else if rr.Type == "CNAME" {
					cnames[rr.Value] = struct{}{}
				}
			}
		}

		for cname := range cnames {
			return resolveRecu(r, cname)
			//out = append(out, resolveRecu(r, cname)...)
		}

		return nil, fmt.Errorf("no host found: %s", host)
	}

	return resolveRecu(r, host)
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

		if strings.Contains(ipAddr.IP.String(), ":") {
			webClient.connTarget = fmt.Sprintf("[%s]:%s", ipAddr.IP.String(), port)
		} else {
			webClient.connTarget = fmt.Sprintf("%s:%s", ipAddr.IP.String(), port)
		}
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

	if webClient.config.DisableHTTP2 {
		webClient.httpClient.Transport.(*http.Transport).TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
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
		Compressed:   !res.Uncompressed,

		IsFailure:    failed,
		FailureCause: failureCause,
	}

}
