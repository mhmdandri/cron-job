package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type SchedulerManager struct {
	cronInst    *cron.Cron
	cronService *CronService
	entryID     cron.EntryID
	schedule    string
	mutex       sync.RWMutex
	nextRun     time.Time
}

func NewSchedulerManager(cronService *CronService, initialSchedule string) (*SchedulerManager, error) {
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	sm := &SchedulerManager{
		cronInst:    c,
		cronService: cronService,
		schedule:    initialSchedule,
	}

	if err := sm.setSchedule(initialSchedule); err != nil {
		return nil, err
	}

	c.Start()
	return sm, nil
}

// Convert shorthand like "15m" or "30m" or "1h" to cron spec "@every 15m" or standard cron "*/15 * * * *"
func parseScheduleSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "*/15 * * * *"
	}

	// If user inputs e.g. "15m", "30m", "15", "30"
	if spec == "15" || spec == "15m" {
		return "*/15 * * * *"
	}
	if spec == "30" || spec == "30m" {
		return "*/30 * * * *"
	}
	if spec == "60" || spec == "60m" || spec == "1h" {
		return "0 * * * *"
	}

	// If it starts with @ (e.g. @every 15m) or has spaces (cron format), use directly
	if strings.HasPrefix(spec, "@") || strings.Contains(spec, " ") {
		return spec
	}

	// Try duration format e.g. "10m" -> "@every 10m"
	if _, err := time.ParseDuration(spec); err == nil {
		return "@every " + spec
	}

	return spec
}

func (sm *SchedulerManager) setSchedule(newSpec string) error {
	spec := parseScheduleSpec(newSpec)

	// Remove old job if exists
	if sm.entryID != 0 {
		sm.cronInst.Remove(sm.entryID)
	}

	entryID, err := sm.cronInst.AddFunc(spec, func() {
		sm.cronService.Execute("scheduled")
		sm.updateNextRun()
	})

	if err != nil {
		return fmt.Errorf("invalid cron schedule expression '%s': %w", newSpec, err)
	}

	sm.entryID = entryID
	sm.schedule = newSpec
	sm.updateNextRun()

	log.Printf("[SCHEDULER] Updated cron schedule to '%s' (Parsed: '%s')", newSpec, spec)
	return nil
}

func (sm *SchedulerManager) updateNextRun() {
	entry := sm.cronInst.Entry(sm.entryID)
	sm.nextRun = entry.Next
}

func (sm *SchedulerManager) UpdateSchedule(newSpec string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.setSchedule(newSpec)
}

func (sm *SchedulerManager) GetScheduleInfo() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	entry := sm.cronInst.Entry(sm.entryID)
	return map[string]interface{}{
		"schedule":     sm.schedule,
		"parsed_spec":  parseScheduleSpec(sm.schedule),
		"next_run":     entry.Next,
		"prev_run":     entry.Prev,
		"is_scheduled": sm.entryID != 0,
	}
}
