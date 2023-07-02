// cron.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"strings"
	"time"

	"github.com/go-co-op/gocron"
)

func (opts *ConfigOpts) Cron() {
	s := gocron.NewScheduler(time.Local)
	s.TagsUnique()
	cmdLists := opts.ConfigFile.CmdConfigLists
	for listName, config := range cmdLists {
		if config.Name == "" {
			config.Name = listName
		}
		cron := strings.TrimSpace(config.Cron)
		if cron != "" {
			opts.ConfigFile.Logger.Info().Str("Scheduling cron list", config.Name).Str("Time", cron).Send()
			_, err := s.CronWithSeconds(cron).Tag(config.Name).Do(func(cron string) {
				opts.ConfigFile.RunListConfig(cron, opts)
			}, cron)
			if err != nil {
				panic(err)
			}
		}
	}
	opts.ConfigFile.Logger.Info().Msg("Starting cron mode...")
	s.StartBlocking()
}
