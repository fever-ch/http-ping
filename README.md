[![Go Report Card](https://goreportcard.com/badge/github.com/fever-ch/http-ping)](https://goreportcard.com/report/github.com/fever-ch/http-ping)
[![pr/push checks](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml/badge.svg)](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml)
[![MIT license](https://img.shields.io/badge/license-Apache-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/fever-ch/http-ping)

# Http-Ping

`http-ping` is a free software distributed under the [Apache License 2.0](LICENSE).

This piece of software is similar to the usual [_ping networking utility_](https://en.wikipedia.org/wiki/Ping_(networking_utility)) but instead of working on top of ICMP`, it works on top of
HTTP/S.

is a small, free, easy-to-use command line utility that probes a given URL and displays relevant statistics. It is similar to the popular ping utility, but works over HTTP/S instead of ICMP, and with a URL instead of a computer name/IP address. http-ping supports IPv6 addresses.

## Platforms

This software is written in [Go](https://go.dev), and should then benefit from the [wide list of targets provided by Go](https://go.dev/doc/install/source#environment).

This software has been reported to work well on:
- *Linux:* amd64, 386, arm64, arm
- *Windows:* amd64, 386
- *MacOS:* amd64 (Intel Macs), arm64 (Apple Silicon)

## Usage

Simply type `http-ping -h` to get the list of available commands

```
shell> http-ping -h
An utility which evaluates the latency of HTTP/S requests

Usage:
  http-ping [flags] target-URL

Flags:
      --conn-target string      force connection to be done with a specific IP:port (i.e. 127.0.0.1:8080)
      --cookie stringArray      add one or more cookies, in the form name:value
  -c, --count int               define the number of request to be sent (default unlimited)
  -K, --disable-keepalive       disable keep-alive feature
  -x, --extra-parameter         extra changing parameter, add an extra changing parameter to the request to avoid being cached by reverse proxy
  -H, --head                    perform HTTP HEAD requests instead of GETs
  -h, --help                    help for http-ping
  -k, --insecure                allow insecure server connections when using SSL
  -i, --interval duration       define the wait time between each request (default 1s)
  -4, --ipv4                    force IPv4 resolution for dual-stacked sites
  -6, --ipv6                    force IPv6 resolution for dual-stacked sites
      --no-server-error         ignore server errors (5xx), do not handle them as "lost pings"
      --parameter stringArray   add one or more parameters, in the form name:value
  -q, --quiet                   print less details
      --user-agent string       define a custom user-agent (default "Http-Ping/(devel) (https://github.com/fever-ch/http-ping)")
  -v, --verbose                 print more details
      --version                 version for http-ping
  -w, --wait duration           define the time for a response before timing out (default 1s)
```
Measure the latency with the Google Cloud Zurich region:
```
http-ping https://europe-west6-5tkroniexa-oa.a.run.app/api/ping
```

## Use with Docker
```shell
docker run --rm feverch/http-ping -h
```

Note: images are published as `feverch/http-ping` (Central Docker registry) or `ghcr.io/fever-ch/http-ping` (Github Container registry)