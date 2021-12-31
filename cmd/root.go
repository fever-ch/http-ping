// Copyright 2021 Raphaël P. Barazzutti
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

package cmd

import (
	"errors"
	"fever.ch/http-ping/app"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"math"
	"net"
	"regexp"
	"time"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := prepareRootCmd(app.NewHTTPPing)
	cobra.CheckErr(rootCmd.Execute())
}

// extraConfig are config items that are triggered by cobra, but which are not part of the config of HTTPPing, these
// items need to be copied/adapted to HTTPPing (the runner is doing that)
type extraConfig struct {
	ipv4, ipv6 bool

	head bool

	quiet, verbose bool

	cookies []string

	headers []string

	parameters []string
}

type runner struct {
	config   *app.Config
	xp       *extraConfig
	appLogic func(config *app.Config, stdout io.Writer) (app.HTTPPing, error)
	cmd      *cobra.Command
	args     []string
}

func (runner *runner) run() error {
	loaders := []func() error{
		runner.loadTarget,
		runner.loadTarget,
		runner.loadLog,
		runner.loadNetwork,
		runner.loadDNS,
		runner.loadRest,
	}

	for _, loader := range loaders {
		if err := loader(); err != nil {
			return err
		}
	}

	instance, err := runner.appLogic(runner.config, runner.cmd.OutOrStdout())
	if err != nil {
		return err
	}
	return instance.Run()
}

func (runner *runner) isFlagUsed(name string) bool {
	used := false
	runner.cmd.Flags().Visit(func(f *pflag.Flag) {
		if name == f.Name {
			used = true
		}
	})
	return used
}

func (runner *runner) loadTarget() error {
	if len(runner.args) == 0 {
		_ = runner.cmd.Usage()
		runner.cmd.Println()
		return errors.New("target-URL required")
	} else if len(runner.args) > 1 {
		_ = runner.cmd.Usage()
		runner.cmd.Println()
		return errors.New("too many arguments")
	}

	runner.config.Target = runner.args[0]

	if a, e := regexp.MatchString("^https?://", runner.config.Target); e == nil && !a {
		runner.config.Target = "https://" + runner.config.Target
	}
	return nil
}

func (runner *runner) loadNetwork() error {
	if runner.xp.ipv4 {
		if runner.xp.ipv6 {
			return errors.New("IPv4 and IPv6 cannot be enforced simultaneously")
		}
		runner.config.IPProtocol = "ip4"
	} else if runner.xp.ipv6 {
		runner.config.IPProtocol = "ip6"
	} else {
		runner.config.IPProtocol = "ip"
	}

	return nil
}

func (runner *runner) loadLog() error {
	if runner.xp.verbose {
		if runner.xp.quiet {
			return errors.New("quiet and verbose cannot be enforced simultaneously")
		}
		runner.config.LogLevel = 2
	} else if runner.xp.quiet {
		runner.config.LogLevel = 0
	} else {
		runner.config.LogLevel = 1
	}
	return nil
}

func (runner *runner) loadDNS() error {

	if runner.config.FullDNS && runner.config.DNSServer != "" {
		return errors.New("DNS server cannot specified when full DNS resolutions is enabled")
	}

	if runner.config.DNSServer != "" && net.ParseIP(runner.config.DNSServer) == nil {
		return errors.New("DNS server should be an IPv4 or IPv6 address")
	}
	return nil
}

func (runner *runner) loadRest() error {

	if runner.xp.head {
		if runner.isFlagUsed("method") {
			return errors.New("head and method cannot be enforced simultaneously")
		}

		runner.config.Method = "HEAD"
	}

	if runner.config.Count <= 0 {
		return fmt.Errorf("invalid count of requests to be sent `%d'", runner.config.Count)
	}

	if runner.config.Workers <= 0 {
		return fmt.Errorf("invalid number of workers `%d'", runner.config.Workers)
	}

	for _, cookie := range runner.xp.cookies {
		n, v, e := splitPair(cookie)
		if e != nil {
			return fmt.Errorf("cookie: %s", e)
		}

		runner.config.Cookies = append(runner.config.Cookies, app.Cookie{Name: n, Value: v})
	}

	for _, header := range runner.xp.headers {
		n, v, e := splitPair(header)
		if e != nil {
			return fmt.Errorf("header: %s", e)
		}

		runner.config.Headers = append(runner.config.Headers, app.Header{Name: n, Value: v})
	}

	for _, parameter := range runner.xp.parameters {
		n, v, e := splitPair(parameter)
		if e != nil {
			return fmt.Errorf("parameter: %s", e)
		}

		runner.config.Parameters = append(runner.config.Parameters, app.Parameter{Name: n, Value: v})
	}
	return nil
}

func splitPair(str string) (string, string, error) {
	r := regexp.MustCompile("^([[:alnum:]]+)=(.*)$")
	e := r.FindStringSubmatch(str)
	if len(e) == 3 {
		return e[1], e[2], nil
	}
	return "", "", fmt.Errorf("format should be \"key=value\", where key is a non-empty string of alphanumberic characters and value any string, illegal format: \"%s\"", str)
}

func runAndError(config *app.Config, xp *extraConfig, appLogic func(config *app.Config, stdout io.Writer) (app.HTTPPing, error)) func(cmd *cobra.Command, args []string) error {

	return func(cmd *cobra.Command, args []string) error {
		r := runner{appLogic: appLogic, config: config, xp: xp, cmd: cmd, args: args}
		return r.run()
	}
}

func prepareRootCmd(appLogic func(config *app.Config, stdout io.Writer) (app.HTTPPing, error)) *cobra.Command {

	var config = &app.Config{}

	xp := &extraConfig{}

	var rootCmd = &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,

		Use: "http-ping [flags] target-URL",

		Short: "An utility which evaluates the latency of HTTP/S requests",
		Long:  `An utility which evaluates the latency of HTTP/S requests`,

		Version: app.Version,
		RunE:    runAndError(config, xp, appLogic),
	}

	rootCmd.Flags().StringVar(&config.UserAgent, "user-agent", fmt.Sprintf("Http-Ping/%s (%s)", app.Version, app.ProjectURL), "define a custom user-agent")

	rootCmd.Flags().StringVarP(&config.ConnTarget, "conn-target", "", "", "force connection to be done with a specific IP:port (i.e. 127.0.0.1:8080)")

	rootCmd.Flags().StringVarP(&config.Method, "method", "", "GET", "select a which HTTP method to be used")

	rootCmd.Flags().BoolVarP(&xp.head, "head", "H", false, "perform HTTP HEAD requests instead of GETs")

	rootCmd.Flags().BoolVarP(&xp.ipv4, "ipv4", "4", false, "force IPv4 resolution for dual-stacked sites")

	rootCmd.Flags().BoolVarP(&xp.ipv6, "ipv6", "6", false, "force IPv6 resolution for dual-stacked sites")

	rootCmd.Flags().BoolVarP(&config.DisableKeepAlive, "disable-keepalive", "K", false, "disable keep-alive feature")

	rootCmd.Flags().DurationVarP(&config.Wait, "wait", "w", 10*time.Second, "define the time for a response before timing out")

	rootCmd.Flags().DurationVarP(&config.Interval, "interval", "i", 1*time.Second, "define the wait time between each request")

	rootCmd.Flags().Int64VarP(&config.Count, "count", "c", math.MaxInt64, "define the number of request to be sent")

	rootCmd.Flag("count").DefValue = "unlimited"

	rootCmd.Flags().BoolVarP(&xp.verbose, "verbose", "v", false, "print more details")

	rootCmd.Flags().BoolVarP(&xp.quiet, "quiet", "q", false, "print less details")

	rootCmd.Flags().BoolVarP(&config.NoCheckCertificate, "insecure", "k", false, "allow insecure server connections when using SSL")

	rootCmd.Flags().StringArrayVarP(&xp.cookies, "cookie", "", []string{}, "add one or more cookies, in the form name=value")

	rootCmd.Flags().StringArrayVarP(&xp.headers, "header", "", []string{}, "add one or more header, in the form name=value")

	rootCmd.Flags().StringArrayVarP(&xp.parameters, "parameter", "", []string{}, "add one or more parameters to the query, in the form name:value")

	rootCmd.Flags().BoolVarP(&config.IgnoreServerErrors, "no-server-error", "", false, "ignore server errors (5xx), do not handle them as \"lost pings\"")

	rootCmd.Flags().BoolVarP(&config.ExtraParam, "extra-parameter", "x", false, "extra changing parameter, add an extra changing parameter to the request to avoid being cached by reverse proxy")

	rootCmd.Flags().BoolVarP(&config.DisableCompression, "disable-compression", "", false, "the client will not request the remote server to compress answers (hence it might actually do it)")

	rootCmd.Flags().BoolVarP(&config.AudibleBell, "audible-bell", "a", false, "audible ; include a bell (ASCII 0x07) character in the output when any successful answer is received")

	rootCmd.Flags().StringVarP(&config.Referrer, "referrer", "", "", "define the referrer")

	rootCmd.Flags().StringVarP(&config.AuthUsername, "auth-username", "", "", "authentication username")

	rootCmd.Flags().StringVarP(&config.AuthPassword, "auth-password", "", "", "authentication password")

	rootCmd.Flags().BoolVarP(&config.DisableHTTP2, "disable-http2", "", false, "disable the HTTP/2 protocol")

	rootCmd.Flags().BoolVarP(&config.FullDNS, "dns-full-resolution", "D", false, "enable full DNS resolution from the root servers")

	rootCmd.Flags().StringVarP(&config.DNSServer, "dns-server", "d", "", "specify an alternate DNS server for resolutions")

	rootCmd.Flags().BoolVarP(&config.CacheDNSRequests, "dns-cache", "", false, "cache DNS requests")

	rootCmd.Flags().BoolVarP(&config.KeepCookies, "keep-cookies", "", false, "keep received cookies between requests")

	rootCmd.Flags().BoolVarP(&config.FollowRedirects, "follow-redirects", "F", false, "follow HTTP redirects (codes 3xx)")

	rootCmd.Flags().IntVarP(&config.Workers, "workers", "", 1, "define the number of workers to be used")

	rootCmd.Flags().BoolVarP(&config.Tput, "throughput", "t", false, "log the number of requests done per second")

	return rootCmd
}
