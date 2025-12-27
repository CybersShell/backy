// cron.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

var defaultPort = 8888

func (opts *ConfigOpts) Cron() {
	s, _ := gocron.NewScheduler(gocron.WithLocation(time.Local))
	defer func() { _ = s.Shutdown() }()
	opts.Logger.Info().Msg("Starting cron mode...")
	s.Start()
	cmdLists := opts.CmdConfigLists
	for _, config := range cmdLists {
		cron := strings.TrimSpace(config.Cron)
		if cron != "" {
			job, err := s.NewJob(
				gocron.CronJob(cron, opts.GoCron.UseSeconds),
				gocron.NewTask(
					func(cronStr string) {
						opts.RunListConfig(cronStr)
					},
					cron,
				),
				gocron.WithName(config.Name))
			if err != nil {
				logging.ExitWithMSG(fmt.Sprintf("error: %v", err), 1, &opts.Logger)
			}
			nextRun, _ := job.NextRun()
			opts.Logger.Info().Str("Scheduling cron list", config.Name).Str("Time", cron).Str("Next run", nextRun.String()).Send()
		}
	}

	// start the web UI server
	if opts.GoCron.BindAddress == "" {
		if opts.GoCron.Port == 0 {
			opts.GoCron.BindAddress = ":8888"
		} else {
			opts.GoCron.BindAddress = fmt.Sprintf(":%d", opts.GoCron.Port)
		}
	} else {
		if opts.GoCron.Port != 0 {
			opts.GoCron.BindAddress = fmt.Sprintf("%s:%d", opts.GoCron.BindAddress, opts.GoCron.Port)
		} else {
			opts.GoCron.BindAddress = fmt.Sprintf("%s:%d", opts.GoCron.BindAddress, defaultPort)
		}
	}
	// consensus := externalip.DefaultConsensus(nil, nil)

	// By default Ipv4 or Ipv6 is returned,
	// use the function below to limit yourself to IPv4,
	// or pass in `6` instead to limit yourself to IPv6.
	// consensus.UseIPProtocol(4)

	// Get your IP,
	// which is never <nil> when err is <nil>.
	// ip, err := consensus.ExternalIP()
	// if err == nil {
	// 	fmt.Println(ip.String()) // print IPv4/IPv6 in string format
	// }
	srv := server.NewServer(s, opts.GoCron.Port)
	// srv := server.NewServer(scheduler, 8080, server.WithTitle("My Custom Scheduler")) // with custom title if you want to customize the title of the UI (optional)
	opts.Logger.Info().Msgf("GoCron UI available at http://%s", opts.GoCron.BindAddress)
	opts.Logger.Fatal().Msg(http.ListenAndServe(opts.GoCron.BindAddress, srv.Router).Error())
	select {} // wait forever
}
