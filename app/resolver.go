// Copyright 2022-2023 - Raphaël P. Barazzutti
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"fmt"
	"github.com/domainr/dnsr"
	"github.com/miekg/dns"
	"net"
	"strings"
)

type resolver struct {
	config      *Config
	cache       map[string]*net.IPAddr
	dnsResolver *dnsr.Resolver
}

func newResolver(config *Config) *resolver {
	return &resolver{
		config:      config,
		cache:       make(map[string]*net.IPAddr),
		dnsResolver: dnsr.NewResolver(dnsr.WithCache(1024)),
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

func resolveWithSpecificServerQtype(qtype uint16, server string, host string) ([]*net.IP, error) {
	var ips []*net.IP

	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = []dns.Question{}

	msg.Question = append(msg.Question, dns.Question{Name: host, Qtype: qtype, Qclass: dns.ClassINET})

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
		ip    []*net.IP
		err   error
		qtype uint16
	}

	if network == "ip4" {
		return resolveWithSpecificServerQtype(dns.TypeA, server, host)
	} else if network == "ip6" {
		return resolveWithSpecificServerQtype(dns.TypeAAAA, server, host)
	} else {
		var ipv4Address []*net.IP = nil

		answersChan := make(chan *resolveAnswer)

		ret := func(qtype uint16) {
			out, err := resolveWithSpecificServerQtype(qtype, server, host)

			answersChan <- &resolveAnswer{out, err, qtype}
		}

		go ret(dns.TypeAAAA)
		go ret(dns.TypeA)

		for i := 0; i < 2; i++ {
			if answer := <-answersChan; answer.err == nil && len(answer.ip) > 0 {

				if answer.qtype == dns.TypeAAAA {
					return answer.ip, nil
				}

				ipv4Address = append(ipv4Address, answer.ip...)
			}

		}

		if ipv4Address == nil {
			return nil, noSuchHostError(host)
		}
		return ipv4Address, nil
	}
}

func noSuchHostError(host string) error {
	return &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
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
			return nil, noSuchHostError(addr)
		}
		return &net.IPAddr{IP: ip}, nil
	} else if resolver.config.DNSServer != "" {
		ip, err := resolveWithSpecificServer(resolver.config.IPProtocol, resolver.config.DNSServer, fmt.Sprintf("%s.", addr))
		if err != nil {
			return nil, err
		}

		if len(ip) == 0 {
			return nil, noSuchHostError(addr)
		}

		return &net.IPAddr{IP: *ip[0]}, nil
	} else {
		return net.ResolveIPAddr(resolver.config.IPProtocol, addr)
	}
}

func (resolver *resolver) fullResolveFromRoot(network, host string) (*string, error) {
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

	return resolver.resolveRecu(host, qtypes)
}

func (resolver *resolver) resolveRecu(host string, qtypes []string) (*string, error) {

	cnames := make(map[string]struct{})
	for _, qtype := range qtypes {
		for _, rr := range resolver.dnsResolver.Resolve(host, qtype) {
			if rr.Type == qtype {
				return &rr.Value, nil
			} else if rr.Type == "CNAME" {
				cnames[rr.Value] = struct{}{}
			}
		}
	}

	for cname := range cnames {
		return resolver.resolveRecu(cname, qtypes)
	}

	return nil, fmt.Errorf("no host found: %s", host)
}
