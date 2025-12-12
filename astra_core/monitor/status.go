package monitor

import (
	"astra_core/notifier"
	"astra_core/steam"
	"log"
	"time"
)

type StatusMonitor struct {
	webClient *steam.SteamWebClient
	notifier  *notifier.DiscordNotifier

	lastSteamStatus string
	lastCS2Status   string
}

func NewStatusMonitor(notifier *notifier.DiscordNotifier) *StatusMonitor {
	return &StatusMonitor{
		webClient:       steam.NewSteamWebClient(),
		notifier:        notifier,
		lastSteamStatus: "online",
		lastCS2Status:   "online",
	}
}

func (s *StatusMonitor) Start() {
	go func() {
		log.Println("Starting Status Monitor...")
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			s.check()
		}
	}()
}

func (s *StatusMonitor) check() {
	status, err := s.webClient.GetServerStatus()
	if err != nil {
		log.Printf("StatusMonitor: Error checking status: %v", err)
		return
	}

	isMaintenance := IsMaintenanceWindow()

	s.evaluate("Steam", &s.lastSteamStatus, status.Steam, isMaintenance)
	s.evaluate("CS2", &s.lastCS2Status, status.CS2, isMaintenance)
}

func (s *StatusMonitor) evaluate(serviceName string, lastStatus *string, currentStatus string, isMaintenance bool) {
	if currentStatus == *lastStatus {
		return
	}

	log.Printf("Status Change Detected for %s: %s -> %s", serviceName, *lastStatus, currentStatus)

	if s.notifier != nil {
		update := notifier.StatusUpdate{
			Service:       serviceName,
			OldStatus:     *lastStatus,
			NewStatus:     currentStatus,
			IsMaintenance: isMaintenance,
		}

		// Only notify if status is meaningful (ignore 'unknown' if previous was online unless persistent, or simplify)
		if currentStatus != "unknown" {
			if err := s.notifier.NotifyStatus(update); err != nil {
				log.Printf("Failed to notify status for %s: %v", serviceName, err)
			}
		}
	}

	*lastStatus = currentStatus
}

// IsMaintenanceWindow checks if current time is Tuesday evening (approx 16:00 - 19:00 EST)
// UTC: Tuesday 21:00 - Wednesday 00:00 (approx)
func IsMaintenanceWindow() bool {
	now := time.Now().UTC()

	if now.Weekday() == time.Tuesday {
		hour := now.Hour()
		// Maintenance usually happens around 23:00 UTC +/- 2 hours
		if hour >= 21 || hour <= 23 {
			return true
		}
	}
	return false
}
