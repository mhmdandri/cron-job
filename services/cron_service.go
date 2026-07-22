package services

import (
	"context"

	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"cron-job/config"
)

type ExecutionLog struct {
	ID           int       `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	StatusCode   int       `json:"status_code"`
	DurationMs   int64     `json:"duration_ms"`
	ResponseBody string    `json:"response_body"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	TriggerType  string    `json:"trigger_type"` // "scheduled" or "manual"
}

type CronService struct {
	cfg       *config.Config
	client    *http.Client
	logs      []ExecutionLog
	logMutex  sync.RWMutex
	logIDSeq  int
	lastRun   time.Time
	runCount  int64
	failCount int64
}

func NewCronService(cfg *config.Config) *CronService {
	return &CronService{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.HTTPTimeout) * time.Second,
		},
		logs: make([]ExecutionLog, 0),
	}
}

func (s *CronService) Execute(triggerType string) ExecutionLog {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.HTTPTimeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.TargetURL, nil)
	execLog := ExecutionLog{
		Timestamp:   start,
		TriggerType: triggerType,
	}

	if err != nil {
		execLog.Success = false
		execLog.Error = "Failed to create request: " + err.Error()
		log.Printf("[CRON ERROR] %s: %v", triggerType, err)
		s.saveLog(execLog)
		return execLog
	}

	// Set Authorization Bearer header
	if s.cfg.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.BearerToken)
	}
	req.Header.Set("User-Agent", "ShipmentCronMonitor/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	duration := time.Since(start).Milliseconds()
	execLog.DurationMs = duration

	if err != nil {
		execLog.Success = false
		execLog.Error = "HTTP request failed: " + err.Error()
		log.Printf("[CRON ERROR] [%s] Target: %s | Error: %v | Duration: %dms", triggerType, s.cfg.TargetURL, err, duration)
		s.saveLog(execLog)
		return execLog
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4096)) // Read max 4KB of response
	bodyStr := ""
	if err == nil {
		bodyStr = string(bodyBytes)
	}

	execLog.StatusCode = resp.StatusCode
	execLog.ResponseBody = bodyStr
	execLog.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	if execLog.Success {
		log.Printf("[CRON SUCCESS] [%s] Status: %d | Duration: %dms", triggerType, resp.StatusCode, duration)
	} else {
		log.Printf("[CRON WARN] [%s] Status: %d | Duration: %dms | Response: %s", triggerType, resp.StatusCode, duration, bodyStr)
	}

	s.saveLog(execLog)
	return execLog
}

func (s *CronService) saveLog(l ExecutionLog) {
	s.logMutex.Lock()
	defer s.logMutex.Unlock()

	s.logIDSeq++
	l.ID = s.logIDSeq

	s.lastRun = l.Timestamp
	s.runCount++
	if !l.Success {
		s.failCount++
	}

	// Prepend to keep newest logs first
	s.logs = append([]ExecutionLog{l}, s.logs...)

	// Keep only last 50 entries
	if len(s.logs) > 50 {
		s.logs = s.logs[:50]
	}
}

func (s *CronService) GetLogs() []ExecutionLog {
	s.logMutex.RLock()
	defer s.logMutex.RUnlock()

	result := make([]ExecutionLog, len(s.logs))
	copy(result, s.logs)
	return result
}

func (s *CronService) GetStats() map[string]interface{} {
	s.logMutex.RLock()
	defer s.logMutex.RUnlock()

	return map[string]interface{}{
		"target_url": s.cfg.TargetURL,
		"last_run":   s.lastRun,
		"run_count":  s.runCount,
		"fail_count": s.failCount,
	}
}
