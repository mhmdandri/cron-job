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
	enabled     bool
}

func NewSchedulerManager(cronService *CronService, initialSchedule string) (*SchedulerManager, error) {
	c := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
	)))

	sm := &SchedulerManager{
		cronInst:    c,
		cronService: cronService,
		schedule:    initialSchedule,
		enabled:     true,
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
		sm.entryID = 0
	}

	sm.schedule = newSpec

	if !sm.enabled {
		sm.nextRun = time.Time{}
		log.Printf("[SCHEDULER] Updated cron schedule spec to '%s', but cron is currently DISABLED", newSpec)
		return nil
	}

	entryID, err := sm.cronInst.AddFunc(spec, func() {
		sm.cronService.Execute("scheduled")
		sm.updateNextRun()
	})

	if err != nil {
		return fmt.Errorf("invalid cron schedule expression '%s': %w", newSpec, err)
	}

	sm.entryID = entryID
	sm.updateNextRun()

	log.Printf("[SCHEDULER] Updated cron schedule to '%s' (Parsed: '%s')", newSpec, spec)
	return nil
}

func (sm *SchedulerManager) updateNextRun() {
	if sm.entryID == 0 {
		sm.nextRun = time.Time{}
		return
	}
	entry := sm.cronInst.Entry(sm.entryID)
	sm.nextRun = entry.Next
}

func (sm *SchedulerManager) UpdateSchedule(newSpec string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.setSchedule(newSpec)
}

func (sm *SchedulerManager) SetEnabled(enable bool) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if sm.enabled == enable {
		return nil
	}

	sm.enabled = enable

	if enable {
		log.Printf("[SCHEDULER] Cron job ENABLED")
		return sm.setSchedule(sm.schedule)
	} else {
		log.Printf("[SCHEDULER] Cron job DISABLED")
		if sm.entryID != 0 {
			sm.cronInst.Remove(sm.entryID)
			sm.entryID = 0
		}
		sm.nextRun = time.Time{}
		return nil
	}
}

func (sm *SchedulerManager) ToggleEnabled() (bool, error) {
	sm.mutex.Lock()
	targetState := !sm.enabled
	sm.mutex.Unlock()

	err := sm.SetEnabled(targetState)
	return targetState, err
}

func (sm *SchedulerManager) GetScheduleInfo() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var nextRun time.Time
	var prevRun time.Time
	if sm.enabled && sm.entryID != 0 {
		entry := sm.cronInst.Entry(sm.entryID)
		nextRun = entry.Next
		prevRun = entry.Prev
	}

	return map[string]interface{}{
		"schedule":     sm.schedule,
		"parsed_spec":  parseScheduleSpec(sm.schedule),
		"next_run":     nextRun,
		"prev_run":     prevRun,
		"is_scheduled": sm.entryID != 0,
		"enabled":      sm.enabled,
	}
}
