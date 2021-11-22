[![Go Report Card](https://goreportcard.com/badge/github.com/fever-ch/http-ping)](https://goreportcard.com/report/github.com/fever-ch/http-ping)
[![pr/push checks](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml/badge.svg)](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml)

# Http-Ping

`http-ping` is a free software distributed under the [Apache License 2.0](LICENSE).

This piece of software is similar to the usual [_ping networking utility_](https://en.wikipedia.org/wiki/Ping_(networking_utility)) but instead of working on top of ICMP`, it works on top of
HTTP/S.

is a small, free, easy-to-use command line utility that probes a given URL and displays relevant statistics. It is similar to the popular ping utility, but works over HTTP/S instead of ICMP, and with a URL instead of a computer name/IP address. http-ping supports IPv6 addresses.

## Platforms

This software is written in [Go](https://go.dev), and should then benefit from the [wide list of targets provided by Go](https://go.dev/doc/install/source#environment).

This software has been reported to work well on:
- *Linux* (amd64, 386, arm64, arm)
- *Windows* (amd64, 386)
- *MacOS* (amd64)

## Usage

Simply type `http-ping -h`

```
An utility which evaluates the latency of HTTP(S) requests

Usage:
  http-ping [flags] target-URL

Flags:
  -c, --count int           define the number of request to be sent (default unlimited)
      --head                perform HTTP HEAD requests instead of GETs
  -h, --help                help for http-ping
  -i, --interval duration   define the wait time between each request (default 1s)
  -4, --ipv4                force IPv4 resolution for dual-stacked sites
  -6, --ipv6                force IPv6 resolution for dual-stacked sites
  -r, --reset-connection    reset connection between requests; ignores keep-alive
      --user-agent string   define a custom user-agent (default "HttpPing/0.1.0 (https://github.com/rbarazzutti/http-ping)")
  -v, --version             version for http-ping
  -w, --wait duration       define the time for a response before timing out (default 1s)

```
