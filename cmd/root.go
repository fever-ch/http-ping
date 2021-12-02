package cmd

import (
	"errors"
	"fmt"
	"github.com/fever-ch/http-ping/app"
	"github.com/spf13/cobra"
	"math"
	"time"
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := prepareRootCmd()
	cobra.CheckErr(rootCmd.Execute())
}

func prepareRootCmd() *cobra.Command {

	var config = cmdConfig{}

	var rootCmd = &cobra.Command{
		SilenceUsage:  true,
		SilenceErrors: true,

		Use: "http-ping [flags] target-URL",

		Short: "An utility which evaluates the latency of HTTP/S requests",
		Long:  `An utility which evaluates the latency of HTTP/S requests`,

		Version: app.Version,

		RunE: func(cmd *cobra.Command, args []string) error {

			if config.ipv4 && config.ipv6 {
				return errors.New("IPv4 and IPv6 cannot be enforced simultaneously")
			}

			if config.quiet && config.verbose {
				return errors.New("quiet and verbose cannot be enforced simultaneously")
			}

			if config.count <= 0 {
				return fmt.Errorf("invalid count of requests to be sent `%d'", config.count)
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

			config.target = args[0]
			app.HTTPPing(&config)

			return nil
		},
	}

	rootCmd.Flags().StringVar(&config.userAgent, "user-agent", fmt.Sprintf("Http-Ping/%s (%s)", app.Version, app.ProjectURL), "define a custom user-agent")

	rootCmd.Flags().StringVar(&config.connTarget, "conn-target", "", "force connection to be done with a specific IP:port (i.e. 127.0.0.1:8080)")

	rootCmd.Flags().BoolVarP(&config.head, "head", "", false, "perform HTTP HEAD requests instead of GETs")

	rootCmd.Flags().BoolVarP(&config.ipv4, "ipv4", "4", false, "force IPv4 resolution for dual-stacked sites")
	rootCmd.Flags().BoolVarP(&config.ipv6, "ipv6", "6", false, "force IPv6 resolution for dual-stacked sites")

	rootCmd.Flags().BoolVarP(&config.fullConnection, "reset-connection", "r", false, "reset connection between requests; ignores keep-alive")

	rootCmd.Flags().DurationVarP(&config.wait, "wait", "w", time.Second, "define the time for a response before timing out")

	rootCmd.Flags().DurationVarP(&config.interval, "interval", "i", 1*time.Second, "define the wait time between each request")

	rootCmd.Flags().Int64VarP(&config.count, "count", "c", math.MaxInt, "define the number of request to be sent")
	rootCmd.Flag("count").DefValue = "unlimited"

	rootCmd.Flags().BoolVarP(&config.verbose, "verbose", "v", false, "print more details")

	rootCmd.Flags().BoolVarP(&config.quiet, "quiet", "q", false, "print less details")

	return rootCmd
}
