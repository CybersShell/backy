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
	for _, config := range conf.CmdConfigLists {
		if strings.TrimSpace(config.Cron) != "" {
			_, err := s.CronWithSeconds(config.Cron).Tag(config.Name).Do(func(cron string) {
				conf.RunBackyConfig(cron)
			}, config.Cron)
			if err != nil {
				panic(err)
			}
		}
	}
	s.StartBlocking()
}
