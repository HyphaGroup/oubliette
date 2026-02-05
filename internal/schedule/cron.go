package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

// cronParser is configured for standard 5-field cron (minute hour day month weekday)
var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ParseCron validates and parses a cron expression
// Returns an error if the expression is invalid
func ParseCron(expr string) (cron.Schedule, error) {
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidCron, err)
	}
	return sched, nil
}

// NextRun calculates the next run time after the given time
// Returns zero time if the expression is invalid
func NextRun(expr string, after time.Time) (time.Time, error) {
	sched, err := ParseCron(expr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(after), nil
}

// ValidateCron checks if a cron expression is valid
func ValidateCron(expr string) error {
	_, err := ParseCron(expr)
	return err
}
