package monitor

import (
	"astra_core/database"
	"astra_core/depot"
	"astra_core/diff"
	"astra_core/extractor"
	"astra_core/notifier"
	"astra_core/steamcmd"
	"encoding/json"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Monitor struct {
	client           *steamcmd.Client
	db               *database.DB
	tracker          *diff.Tracker
	notifier         *notifier.DiscordNotifier
	downloader       *depot.Downloader
	appID            int
	lastChangeNumber string
	lastDiff         *diff.DiffResult
}

func NewMonitor(appID int, db *database.DB, webhookURL string) *Monitor {
	client := steamcmd.NewClient()
	var notif *notifier.DiscordNotifier
	if webhookURL != "" {
		notif = notifier.NewDiscordNotifier(webhookURL)
	}

	return &Monitor{
		client:     client,
		db:         db,
		tracker:    diff.NewTracker(client),
		notifier:   notif,
		downloader: depot.NewDownloader(appID),
		appID:      appID,
	}
}

func (m *Monitor) LoadState() {
	cn, _, _, _, err := m.db.GetAppState(m.appID)
	if err != nil {
		log.Printf("Failed to load state: %v", err)
		return
	}
	m.lastChangeNumber = cn

	diffData, err := m.db.GetLastDiff(m.appID)
	if err != nil {
		log.Printf("Failed to load last diff: %v", err)
		return
	}
	if diffData != nil {
		var loadedDiff diff.DiffResult
		if err := json.Unmarshal(diffData, &loadedDiff); err == nil {
			m.lastDiff = &loadedDiff
			log.Printf("Loaded last diff: Type=%s, Strings=%d", loadedDiff.Type, len(loadedDiff.NewStrings))
		}
	}
}

func (m *Monitor) SaveState(info *steamcmd.AppInfo, rawVDF string) {
	data, err := json.Marshal(info)
	if err != nil {
		log.Printf("Failed to marshal AppInfo: %v", err)
		return
	}

	if err := m.db.UpdateAppState(m.appID, info.ChangeNumber, info.BuildID, string(data), rawVDF); err != nil {
		log.Printf("Failed to save state: %v", err)
	}
	m.lastChangeNumber = info.ChangeNumber
}

func (m *Monitor) Start() {
	log.Println("Starting PICS Monitor...")
	if err := m.client.Start(); err != nil {
		log.Fatalf("Failed to start SteamCMD: %v", err)
	}

	if err := m.client.LoginAnonymous(); err != nil {
		log.Printf("Login failed (might be already logged in or retry needed): %v", err)
	}
	time.Sleep(5 * time.Second)

	m.LoadState()
	log.Printf("Loaded State: ChangeNumber=%s", m.lastChangeNumber)

	for {
		m.check()
		time.Sleep(30 * time.Second)
	}
}

func (m *Monitor) check() {
	log.Println("Checking for updates...")
	m.client.AppInfoUpdate(m.appID)
	time.Sleep(2 * time.Second)

	output, err := m.client.AppInfoPrint(m.appID)
	if err != nil {
		log.Printf("Failed to get app info: %v", err)
		return
	}

	info := steamcmd.ParseAppInfo(output)
	if info.ChangeNumber == "" {
		log.Println("Failed to parse ChangeNumber")
		return
	}

	if info.ChangeNumber != m.lastChangeNumber {
		log.Printf("NEW UPDATE DETECTED! Old: %s, New: %s", m.lastChangeNumber, info.ChangeNumber)

		_, _, oldJson, oldRawVDF, _ := m.db.GetAppState(m.appID)
		var oldInfo steamcmd.AppInfo
		if oldJson != "" {
			json.Unmarshal([]byte(oldJson), &oldInfo)
		} else {
			oldInfo = steamcmd.AppInfo{ChangeNumber: m.lastChangeNumber}
		}

		diffResult := m.tracker.ProcessUpdate(&oldInfo, info)
		diffResult.RawDiff = diff.GenerateUnifiedDiff(oldRawVDF, output, "old", "new")

		m.analyzeDepotChanges(diffResult, &oldInfo, info)

		log.Printf("Diff Result: Type=%s, Reason=%s", diffResult.Type, diffResult.TypeReason)
		m.lastDiff = diffResult

		// Persist the diff result
		if diffData, err := json.Marshal(diffResult); err == nil {
			if err := m.db.SaveLastDiff(m.appID, diffData); err != nil {
				log.Printf("Failed to save last diff: %v", err)
			}
		}

		if m.notifier != nil {
			if err := m.notifier.Notify(diffResult); err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}

		m.handleUpdate(info, output)
	} else {
		log.Printf("No changes. Current: %s", info.ChangeNumber)
	}
}

func (m *Monitor) analyzeDepotChanges(result *diff.DiffResult, oldInfo, newInfo *steamcmd.AppInfo) {
	// Apenas Dedicated Server (734) e Win64 (735) são suficientes para analisar strings e CVars
	binaryDepots := []string{"734", "735"}

	for _, change := range result.ChangedDepots {
		if !contains(binaryDepots, change.ID) {
			continue
		}

		// If OldGID is empty, it means it's a new depot or first run.
		// We still want to analyze it to extract strings.
		log.Printf("Analyzing depot %s (%s)...", change.ID, change.Name)

		m.downloadAndAnalyzeDepot(result, change)
	}
}

func (m *Monitor) downloadAndAnalyzeDepot(result *diff.DiffResult, change diff.DepotChange) {
	m.downloader.CleanupOldCache()

	newPath, err := m.downloader.DownloadDepot(mustAtoi(change.ID), change.NewGID, "")
	if err != nil {
		log.Printf("Failed to download new depot: %v", err)
		return
	}

	var oldPath string
	if change.OldGID != "" {
		oldPath, _ = m.downloader.DownloadDepot(mustAtoi(change.ID), change.OldGID, "")
	}

	m.extractAndCompare(result, oldPath, newPath)
}

func (m *Monitor) extractAndCompare(result *diff.DiffResult, oldPath, newPath string) {
	binaries := []string{"*.exe", "*.dll", "*.so"}
	log.Printf("Starting extraction in %s", newPath)

	for _, pattern := range binaries {
		newFiles, _ := filepath.Glob(filepath.Join(newPath, "**", pattern))
		if len(newFiles) > 0 {
			log.Printf("Found %d files matching %s", len(newFiles), pattern)
		}

		for _, newFile := range newFiles {
			log.Printf("Extracting strings from %s...", filepath.Base(newFile))
			newStrings, err := extractor.ExtractStrings(newFile)
			if err != nil {
				log.Printf("Extraction failed for %s: %v", filepath.Base(newFile), err)
				continue
			}
			log.Printf("Extracted %d strings from %s", len(newStrings), filepath.Base(newFile))

			interesting := extractor.FilterInterestingStrings(newStrings)
			for _, match := range interesting {
				result.NewStrings = append(result.NewStrings, match.Value)
			}

			protos := extractor.ExtractProtobufs(newStrings)
			for _, proto := range protos {
				result.NewProtobufs = append(result.NewProtobufs, proto.Name)
			}

			if oldPath != "" {
				relPath, _ := filepath.Rel(newPath, newFile)
				oldFile := filepath.Join(oldPath, relPath)
				oldStrings, err := extractor.ExtractStrings(oldFile)
				if err == nil {
					added, removed := extractor.CompareStringSets(oldStrings, newStrings)
					m.tracker.EnhanceWithStringAnalysis(result, added, removed)
				}
			}
		}
	}

	result.Analysis = generateAnalysisSummary(result)
}

func generateAnalysisSummary(result *diff.DiffResult) string {
	var sb strings.Builder

	sb.WriteString("## Update Analysis\n\n")
	sb.WriteString("**Type:** " + string(result.Type) + "\n")
	if result.TypeReason != "" {
		sb.WriteString("**Reason:** " + result.TypeReason + "\n")
	}
	sb.WriteString("\n")

	if len(result.NewProtobufs) > 0 {
		sb.WriteString("### New Protobufs\n")
		for i, proto := range result.NewProtobufs {
			if i >= 20 {
				sb.WriteString("... and more\n")
				break
			}
			sb.WriteString("- `" + proto + "`\n")
		}
		sb.WriteString("\n")
	}

	if len(result.NewStrings) > 0 {
		sb.WriteString("### Notable New Strings\n")
		for i, s := range result.NewStrings {
			if i >= 20 {
				sb.WriteString("... and more\n")
				break
			}
			sb.WriteString("- `" + s + "`\n")
		}
	}

	return sb.String()
}

func (m *Monitor) handleUpdate(info *steamcmd.AppInfo, rawVDF string) {
	m.SaveState(info, rawVDF)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func mustAtoi(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

type MonitorState struct {
	ChangeNumber string           `json:"change_number"`
	BuildID      string           `json:"build_id"`
	LastDiff     *diff.DiffResult `json:"last_diff,omitempty"`
}

func (m *Monitor) GetState() MonitorState {
	return MonitorState{
		ChangeNumber: m.lastChangeNumber,
		LastDiff:     m.lastDiff,
	}
}
