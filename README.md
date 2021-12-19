[![Go Report Card](https://goreportcard.com/badge/github.com/fever-ch/http-ping)](https://goreportcard.com/report/github.com/fever-ch/http-ping)
[![pr/push checks](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml/badge.svg)](https://github.com/fever-ch/http-ping/actions/workflows/continuous-integration.yml)
[![MIT license](https://img.shields.io/badge/license-Apache-brightgreen.svg)](https://opensource.org/licenses/Apache-2.0)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/fever-ch/http-ping)

# Http-Ping

`http-ping` is a free software distributed under the [Apache License 2.0](LICENSE).

This piece of software is similar to the usual [_ping networking utility_](https://en.wikipedia.org/wiki/Ping_(networking_utility)) but instead of working on top of ICMP, it works on top of
HTTP/S.

Http-Ping is a small, free, easy-to-use command line utility that probes a given URL and displays relevant statistics. It is similar to the popular ping utility, but works over HTTP/S instead of ICMP, and with a URL instead of a computer name/IP address. http-ping supports IPv6 addresses.

## Platforms

This software is written in [Go](https://go.dev), and should then benefit from the [wide list of targets provided by Go](https://go.dev/doc/install/source#environment).

This software has been reported to work well on:
- *Linux:* amd64, 386, arm64, arm
- *Windows:* amd64, 386, arm64
- *MacOS:* amd64 (Intel Macs), arm64 (Apple Silicon)

## Usage

Simply type `http-ping -h` to get the list of available commands

```
> http-ping -h
An utility which evaluates the latency of HTTP/S requests

Usage:
  http-ping [flags] target-URL

Flags:
  -a, --audible-bell            audible ; include a bell (ASCII 0x07) character in the output when any successful answer is received
      --auth-password string    authentication username
      --auth-username string    authentication username
      --conn-target string      force connection to be done with a specific IP:port (i.e. 127.0.0.1:8080)
      --cookie stringArray      add one or more cookies, in the form name=value
  -c, --count int               define the number of request to be sent (default unlimited)
      --disable-compression     the client will not request the remote server to compress answers (hence it might actually do it)
      --disable-http2           disable the HTTP/2 protocol
  -K, --disable-keepalive       disable keep-alive feature
      --dns-cache               cache DNS requests
  -D, --dns-full-resolution     enable full DNS resolution from the root servers
  -d, --dns-server string       specify an alternate DNS server for resolutions
  -x, --extra-parameter         extra changing parameter, add an extra changing parameter to the request to avoid being cached by reverse proxy
  -H, --head                    perform HTTP HEAD requests instead of GETs
      --header stringArray      add one or more header, in the form name=value
  -h, --help                    help for http-ping
  -k, --insecure                allow insecure server connections when using SSL
  -i, --interval duration       define the wait time between each request (default 1s)
  -4, --ipv4                    force IPv4 resolution for dual-stacked sites
  -6, --ipv6                    force IPv6 resolution for dual-stacked sites
      --keep-cookies            keep received cookies between requests
      --no-server-error         ignore server errors (5xx), do not handle them as "lost pings"
      --parameter stringArray   add one or more parameters to the query, in the form name:value
  -q, --quiet                   print less details
      --referrer string         define the referrer
      --user-agent string       define a custom user-agent (default "Http-Ping/(devel) (https://github.com/fever-ch/http-ping)")
  -v, --verbose                 print more details
      --version                 version for http-ping
  -w, --wait duration           define the time for a response before timing out (default 1s)
```
Measure the latency with the Google Cloud Zurich region with 4 HTTP pings (`-c 4`):
```
> http-ping https://europe-west6-5tkroniexa-oa.a.run.app/api/ping -c 4
HTTP-PING https://europe-west6-5tkroniexa-oa.a.run.app/api/ping GET

       0: 216.239.36.53:443, code=200, size=13 bytes, time=17.9 ms
       1: 216.239.36.53:443, code=200, size=13 bytes, time=16.7 ms
       2: 216.239.36.53:443, code=200, size=13 bytes, time=16.4 ms
       3: 216.239.36.53:443, code=200, size=13 bytes, time=17.6 ms

--- https://europe-west6-5tkroniexa-oa.a.run.app/api/ping ping statistics ---
4 requests sent, 4 answers received, 0.0% loss
round-trip min/avg/max/stddev = 16.401/17.144/17.915/0.625 ms
```

Measure the latency with Google Cloud Zurich with a single HTTP ping (`-c 1`), disabling socket reuse (`-K`), using a HEAD request (`-H`), and in verbose mode (`-v`):
```
> http-ping https://europe-west6-5tkroniexa-oa.a.run.app/api/ping -c 1 -K -H -v
HTTP-PING https://europe-west6-5tkroniexa-oa.a.run.app/api/ping HEAD

       0: 216.239.36.53:443, code=200, size=0 bytes, time=53.2 ms
          proto=HTTP/2.0, socket reused=false, compressed=true
          network i/o: bytes read=4713, bytes written=671
          tls version=TLS-1.3

          latency contributions:
            53.2 ms request and response
                     ├─   34.8 ms connection setup
                     │             ├─    1.6 ms DNS resolution
                     │             ├─    9.5 ms TCP handshake
                     │             └─   23.5 ms TLS handshake
                     ├─    0.1 ms request sending
                     ├─   17.9 ms wait
                     └─    0.2 ms response ingestion

--- https://europe-west6-5tkroniexa-oa.a.run.app/api/ping ping statistics ---
1 requests sent, 1 answers received, 0.0% loss
round-trip min/avg/max/stddev = 53.250/53.250/53.250/0.000 ms
```
_note: the latency contribution tree only covers the main steps of the HTTP exchange, thus the sum doesn't fully match._
## Use with Docker
```shell
docker run --rm feverch/http-ping
```

Note: images are published as `feverch/http-ping` (Central Docker registry) or `ghcr.io/fever-ch/http-ping` (Github Container registry)