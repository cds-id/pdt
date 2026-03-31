// backend/internal/scheduler/cron.go
package scheduler

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func NextCronRun(expr string, after time.Time) (time.Time, error) {
	schedule, err := cronParser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return schedule.Next(after), nil
}

func NextRunAt(triggerType, cronExpr string, intervalSeconds int, after time.Time) (*time.Time, error) {
	switch triggerType {
	case "cron":
		next, err := NextCronRun(cronExpr, after)
		if err != nil {
			return nil, err
		}
		return &next, nil
	case "interval":
		next := after.Add(time.Duration(intervalSeconds) * time.Second)
		return &next, nil
	case "once":
		return nil, nil // once runs immediately, no next run
	case "event":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", triggerType)
	}
}
