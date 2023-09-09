// cron.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"fmt"
	"strings"
	"time"

	"git.andrewnw.xyz/CyberShell/backy/pkg/logging"
	"github.com/go-co-op/gocron"
)

func (opts *ConfigOpts) Cron() {
	s := gocron.NewScheduler(time.Local)
	s.TagsUnique()
	cmdLists := opts.CmdConfigLists
	for listName, config := range cmdLists {
		if config.Name == "" {
			config.Name = listName
		}

		cron := strings.TrimSpace(config.Cron)
		if cron != "" {
			opts.Logger.Info().Str("Scheduling cron list", config.Name).Str("Time", cron).Send()
			_, err := s.CronWithSeconds(cron).Tag(config.Name).Do(func(cron string) {
				opts.RunListConfig(cron)
			}, cron)
			if err != nil {
				logging.ExitWithMSG(fmt.Sprintf("error: %v", err), 1, &opts.Logger)
			}
		}
	}
	opts.Logger.Info().Msg("Starting cron mode...")
	s.StartBlocking()
}
