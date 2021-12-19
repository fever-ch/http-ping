package app

import (
	"context"
	"fmt"
	"github.com/domainr/dnsr"
	"net"
	"strings"
)

type resolver struct {
	config *Config
	cache  map[string]*net.IPAddr
}

func newResolver(config *Config) *resolver {
	return &resolver{
		config: config,
		cache:  make(map[string]*net.IPAddr),
	}
}

func (resolver *resolver) resolveConn(addr string) (string, error) {
	if host, port, err := net.SplitHostPort(addr); err != nil {
		return "", err
	} else if resolved, err := resolver.resolve(host); err != nil {
		return "", err
	} else {
		if strings.Contains(resolved.IP.String(), ":") {
			return fmt.Sprintf("[%s]:%s", resolved, port), nil
		}
		return fmt.Sprintf("%s:%s", resolved, port), nil
	}
}

func (resolver *resolver) resolve(addr string) (*net.IPAddr, error) {
	if val, ok := resolver.cache[addr]; ok {
		return val, nil
	}

	resolvedAddr, err := resolver.actualResolve(addr)
	if err != nil {
		return nil, err
	}

	if resolver.config.CacheDNSRequests {
		resolver.cache[addr] = resolvedAddr
	}
	return resolvedAddr, err
}

func (resolver *resolver) actualResolve(addr string) (*net.IPAddr, error) {

	if resolver.config.FullDNS {
		var ip net.IP

		if ip = net.ParseIP(addr); ip == nil {
			if entries, err := resolver.fullResolveFromRoot(resolver.config.IPProtocol, addr); err == nil {
				ip = net.ParseIP(*entries)
			}
		}
		if ip == nil {
			return nil, &net.DNSError{Err: "no such host", Name: addr, IsNotFound: true}
		}
		return &net.IPAddr{IP: ip}, nil
	} else if resolver.config.DNSServer != "" {
		dnsTarget := fmt.Sprintf("%s:53", resolver.config.DNSServer)

		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{}

				return d.DialContext(ctx, network, dnsTarget)
			},
		}

		ip, err := r.LookupIP(context.Background(), resolver.config.IPProtocol, addr)
		if err != nil {
			if dnsError, ok := err.(*net.DNSError); ok {
				dnsError.Server = ""
			}
			return nil, err
		}

		return &net.IPAddr{IP: ip[0]}, nil

	}
	return net.ResolveIPAddr(resolver.config.IPProtocol, addr)
}

func (*resolver) fullResolveFromRoot(network, host string) (*string, error) {
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
		}

		return nil, fmt.Errorf("no host found: %s", host)
	}

	return resolveRecu(r, host)
}
