package app

import (
	"fmt"
	"github.com/domainr/dnsr"
	"github.com/miekg/dns"
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

func resolveWithSpecificServerInternal(network, server string, host string) ([]*net.IP, error) {
	var ips []*net.IP

	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = []dns.Question{}

	if network == "ip6" {
		msg.Question = append(msg.Question, dns.Question{Name: host, Qtype: dns.TypeAAAA, Qclass: dns.ClassINET})
	} else if network == "ip4" {
		msg.Question = append(msg.Question, dns.Question{Name: host, Qtype: dns.TypeA, Qclass: dns.ClassINET})
	}

	c := new(dns.Client)

	in, _, err := c.Exchange(msg, fmt.Sprintf("%s:53", server))

	if err != nil {
		return nil, err
	}

	for _, a := range in.Answer {
		if ipv4, ok := a.(*dns.A); ok {
			ips = append(ips, &ipv4.A)
		} else if ipv6, ok := a.(*dns.AAAA); ok {
			ips = append(ips, &ipv6.AAAA)
		}
	}
	return ips, nil
}

func resolveWithSpecificServer(network, server string, host string) ([]*net.IP, error) {

	type resolveAnswer struct {
		ip  []*net.IP
		err error
	}
	if network == "ip" {
		var ips []*net.IP

		answersChan := make(chan *resolveAnswer)
		ret := func(prot string) {
			out, err := resolveWithSpecificServerInternal(prot, server, host)

			answersChan <- &resolveAnswer{out, err}

		}
		go ret("ip4")
		go ret("ip6")

		var answers []*resolveAnswer

		oneSucceeded := false
		for i := 0; i < 2; i++ {
			answer := <-answersChan

			if answer.err == nil {
				oneSucceeded = true
			}

			answers = append(answers, answer)

		}

		if !oneSucceeded {
			return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
		}

		for _, answer := range answers {
			if answer.err == nil {
				ips = append(ips, answer.ip...)
			}
		}

		return ips, nil
	}
	return resolveWithSpecificServerInternal(network, server, host)

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
		ip, err := resolveWithSpecificServer(resolver.config.IPProtocol, resolver.config.DNSServer, fmt.Sprintf("%s.", addr))
		if err != nil {
			return nil, err
		}

		if len(ip) == 0 {
			return nil, &net.DNSError{Err: "no such host", Name: addr, IsNotFound: true}
		}

		return &net.IPAddr{IP: *ip[0]}, nil
	} else {
		return net.ResolveIPAddr(resolver.config.IPProtocol, addr)
	}
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
