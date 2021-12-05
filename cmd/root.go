package cmd

import (
	"errors"
	"fmt"
	"github.com/fever-ch/http-ping/app"
	"github.com/spf13/cobra"
	"math"
	"regexp"
	"time"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := prepareRootCmd()
	cobra.CheckErr(rootCmd.Execute())
}

func prepareRootCmd() *cobra.Command {

	var config = app.Config{}

	var ipv4, ipv6 bool

	var head bool

	var quiet, verbose bool

	var cookies []string

	var parameters []string

	var rootCmd = &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,

		Use: "http-ping [flags] target-URL",

		Short: "An utility which evaluates the latency of HTTP/S requests",
		Long:  `An utility which evaluates the latency of HTTP/S requests`,

		Version: app.Version,

		RunE: func(cmd *cobra.Command, args []string) error {

			if ipv4 && ipv6 {
				return errors.New("IPv4 and IPv6 cannot be enforced simultaneously")
			} else if !ipv4 && !ipv6 {
				config.IPProtocol = "ip"
			} else if ipv4 {
				config.IPProtocol = "ip4"
			} else {
				config.IPProtocol = "ip6"
			}

			if verbose && quiet {
				return errors.New("quiet and verbose cannot be enforced simultaneously")
			} else if verbose {
				config.LogLevel = 2
			} else if quiet {
				config.LogLevel = 0
			} else {
				config.LogLevel = 1
			}

			if head {
				config.Method = "HEAD"
			} else {
				config.Method = "GET"
			}

			if a, e := regexp.MatchString("^https?://", config.Target); e == nil && !a {
				config.Target = "https://" + config.Target
			}

			if config.Count <= 0 {
				return fmt.Errorf("invalid count of requests to be sent `%d'", config.Count)
			}

			if len(args) == 0 {
				_ = cmd.Usage()
				println()
				return errors.New("target-URL required")
			} else if len(args) > 1 {
				_ = cmd.Usage()
				println()
				return errors.New("too many arguments")
			}

			for _, cookie := range cookies {
				n, v := splitPair(cookie)
				if n != "" {
					config.Cookies = append(config.Cookies, app.Cookie{Name: n, Value: v})
				}
			}

			for _, parameter := range parameters {
				n, v := splitPair(parameter)
				if n != "" {
					config.Parameters = append(config.Parameters, app.Parameter{Name: n, Value: v})
				}
			}

			config.Target = args[0]
			app.HTTPPing(&config)

			return nil
		},
	}

	rootCmd.Flags().StringVar(&config.UserAgent, "user-agent", fmt.Sprintf("Http-Ping/%s (%s)", app.Version, app.ProjectURL), "define a custom user-agent")

	rootCmd.Flags().StringVarP(&config.ConnTarget, "conn-target", "", "", "force connection to be done with a specific IP:port (i.e. 127.0.0.1:8080)")

	rootCmd.Flags().BoolVarP(&head, "head", "H", false, "perform HTTP HEAD requests instead of GETs")

	rootCmd.Flags().BoolVarP(&ipv4, "ipv4", "4", false, "force IPv4 resolution for dual-stacked sites")

	rootCmd.Flags().BoolVarP(&ipv6, "ipv6", "6", false, "force IPv6 resolution for dual-stacked sites")

	rootCmd.Flags().BoolVarP(&config.DisableKeepAlive, "disable-keepalive", "K", false, "disable keep-alive feature")

	rootCmd.Flags().DurationVarP(&config.Wait, "wait", "w", time.Second, "define the time for a response before timing out")

	rootCmd.Flags().DurationVarP(&config.Interval, "interval", "i", 1*time.Second, "define the wait time between each request")

	rootCmd.Flags().Int64VarP(&config.Count, "count", "c", math.MaxInt, "define the number of request to be sent")

	rootCmd.Flag("count").DefValue = "unlimited"

	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print more details")

	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "print less details")

	rootCmd.Flags().BoolVarP(&config.NoCheckCertificate, "insecure", "k", false, "allow insecure server connections when using SSL")

	rootCmd.Flags().StringArrayVarP(&cookies, "cookie", "", []string{}, "add one or more cookies, in the form name:value")

	rootCmd.Flags().StringArrayVarP(&parameters, "parameter", "", []string{}, "add one or more parameters, in the form name:value")

	rootCmd.Flags().BoolVarP(&config.IgnoreServerErrors, "no-server-error", "", false, "ignore server errors (5xx), do not handle them as \"lost pings\"")

	rootCmd.Flags().BoolVarP(&config.ExtraParam, "extra-parameter", "x", false, "extra changing parameter, add an extra changing parameter to the request to avoid being cached by reverse proxy")

	return rootCmd
}

func splitPair(str string) (string, string) {
	r := regexp.MustCompile("^([^:]*):(.*)$")
	e := r.FindStringSubmatch(str)
	if len(e) == 3 {
		return e[1], e[2]
	}
	return "", ""
}
