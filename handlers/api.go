package handlers

import (
	"net/http"
	"time"

	"cron-job/config"
	"cron-job/services"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	cfg         *config.Config
	cronService *services.CronService
	scheduler   *services.SchedulerManager
}

func NewAPIHandler(cfg *config.Config, cronService *services.CronService, scheduler *services.SchedulerManager) *APIHandler {
	return &APIHandler{
		cfg:         cfg,
		cronService: cronService,
		scheduler:   scheduler,
	}
}

type UpdateScheduleRequest struct {
	Interval string `json:"interval" binding:"required"`
}

type ToggleScheduleRequest struct {
	Enabled *bool `json:"enabled"`
}

func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "shipment-cron-monitor",
	})
}

func (h *APIHandler) GetStatus(c *gin.Context) {
	schedInfo := h.scheduler.GetScheduleInfo()
	stats := h.cronService.GetStats()

	enabled, _ := schedInfo["enabled"].(bool)
	statusStr := "running"
	if !enabled {
		statusStr = "paused"
	}

	nextRunTime, _ := schedInfo["next_run"].(time.Time)
	var secondsRemaining int64 = 0
	if !nextRunTime.IsZero() {
		diff := time.Until(nextRunTime)
		if diff > 0 {
			secondsRemaining = int64(diff.Seconds())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           statusStr,
		"enabled":          enabled,
		"schedule":         schedInfo["schedule"],
		"next_run":         schedInfo["next_run"],
		"next_run_seconds": secondsRemaining,
		"prev_run":         schedInfo["prev_run"],
		"stats":            stats,
		"target":           h.cfg.TargetURL,
	})
}

func (h *APIHandler) GetLogs(c *gin.Context) {
	logs := h.cronService.GetLogs()
	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

func (h *APIHandler) TriggerManual(c *gin.Context) {
	logResult := h.cronService.Execute("manual")
	c.JSON(http.StatusOK, gin.H{
		"message": "Manual trigger executed successfully",
		"result":  logResult,
	})
}

func (h *APIHandler) UpdateSchedule(c *gin.Context) {
	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload. Must provide 'interval'"})
		return
	}

	if err := h.scheduler.UpdateSchedule(req.Interval); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.cfg.CronSchedule = req.Interval

	c.JSON(http.StatusOK, gin.H{
		"message":  "Schedule updated successfully",
		"interval": req.Interval,
		"info":     h.scheduler.GetScheduleInfo(),
	})
}

func (h *APIHandler) ToggleSchedule(c *gin.Context) {
	var req ToggleScheduleRequest
	var newStatus bool

	if err := c.ShouldBindJSON(&req); err == nil && req.Enabled != nil {
		newStatus = *req.Enabled
		if err := h.scheduler.SetEnabled(newStatus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		var err error
		newStatus, err = h.scheduler.ToggleEnabled()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	msg := "Cron schedule activated"
	if !newStatus {
		msg = "Cron schedule paused"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": msg,
		"enabled": newStatus,
		"info":    h.scheduler.GetScheduleInfo(),
	})
}
