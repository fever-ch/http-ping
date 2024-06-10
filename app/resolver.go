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
	"bytes"
	"encoding/base64"
	dns2 "fever.ch/http-ping/net/dns"
	"fmt"
	"github.com/domainr/dnsr"
	"github.com/miekg/dns"
	"net"
	"net/http"
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

func resolveWithSpecificServerQtypes(server string, host string, qtypes []uint16) ([]*net.IP, error) {
	var ips []*net.IP

	msg := new(dns.Msg)
	msg.Id = dns.Id()
	msg.RecursionDesired = true
	msg.Question = []dns.Question{}

	for _, qtype := range qtypes {
		msg.Question = append(msg.Question, dns.Question{Name: host, Qtype: qtype, Qclass: dns.ClassINET})
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
	if network == "ip4" {
		return resolveWithSpecificServerQtypes(server, host, []uint16{dns.TypeA})
	} else if network == "ip6" {
		return resolveWithSpecificServerQtypes(server, host, []uint16{dns.TypeAAAA})
	} else {
		return resolveWithSpecificServerQtypes(server, host, []uint16{dns.TypeAAAA, dns.TypeA})
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
	} else {
		server := resolver.config.DNSServer
		if server == "" {
			hostServers, err := dns2.GetDNSServers()
			if err != nil {
				return nil, err
			}
			server = hostServers[0]
		}

		ip, err := resolveWithSpecificServer(resolver.config.IPProtocol, server, fmt.Sprintf("%s.", addr))
		if err != nil {
			return nil, err
		}

		if len(ip) == 0 {
			return nil, noSuchHostError(addr)
		}

		return &net.IPAddr{IP: *ip[0]}, nil
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

	for cname := range cnames { // todo will leave athe first one
		return resolver.resolveRecu(cname, qtypes)
	}

	return nil, fmt.Errorf("no host found: %s", host)
}

type BackendResolver interface {
	Resolve(host string, qtype []uint16) (*dns.Msg, error)
}

func NewDoHResolver(url string) BackendResolver {
	return &DoHResolver{doHEndpoint: url}
}

type DoHResolver struct {
	doHEndpoint string
}

func (dohResolver *DoHResolver) Resolve(host string, qtypes []uint16) (*dns.Msg, error) {

	msg := new(dns.Msg)
	for _, qtype := range qtypes {
		msg.Question = append(msg.Question, dns.Question{Name: dns.Fqdn(host), Qtype: qtype})
		// shall i add  Qclass: dns.ClassINET?
	}

	// Convert the DNS message to wire format
	wireMsg, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	// Convert the wire format message to base64 URL encoding
	encodedMsg := base64.RawURLEncoding.EncodeToString(wireMsg)

	// Create the DoH request
	req, err := http.NewRequest("GET", dohResolver.doHEndpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("dns", encodedMsg)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("accept", "application/dns-message")

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read and unpack the response
	respMsg := new(dns.Msg)
	respBody := new(bytes.Buffer)
	_, err = respBody.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}
	err = respMsg.Unpack(respBody.Bytes())
	if err != nil {
		return nil, err
	}
	// Parse the response for TLSA records
	return respMsg, nil
}

//  https://cloudflare-dns.com/dns-query
//  https://dns.google/dns-query
