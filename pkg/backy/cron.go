// cron.go
// Copyright (C) Andrew Woodlee 2023
// License: Apache-2.0

package backy

import (
	"strings"
	"time"

	"github.com/go-co-op/gocron"
)

func (conf *BackyConfigFile) Cron() {
	s := gocron.NewScheduler(time.Local)
	s.TagsUnique()
	for listName, config := range conf.CmdConfigLists {
		if config.Name == "" {
			config.Name = listName
		}
		cron := strings.TrimSpace(config.Cron)
		if cron != "" {
			conf.Logger.Info().Str("Scheduling cron list", config.Name).Str("Time", cron).Send()
			_, err := s.CronWithSeconds(cron).Tag(config.Name).Do(func(cron string) {
				conf.RunBackyConfig(cron)
			}, cron)
			if err != nil {
				panic(err)
			}
		}
	}
	conf.Logger.Info().Msg("Starting cron mode...")
	s.StartBlocking()
}
