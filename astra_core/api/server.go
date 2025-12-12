package api

import (
	"astra_core/diff"
	"astra_core/monitor"
	"astra_core/steam"
	"compress/gzip"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type Server struct {
	mon         *monitor.Monitor
	steamClient *steam.SteamWebClient
	startTime   time.Time
}

func NewServer(mon *monitor.Monitor) *Server {
	return &Server{
		mon:         mon,
		steamClient: steam.NewSteamWebClient(),
		startTime:   time.Now(),
	}
}

func (s *Server) Start(addr string) {
	http.HandleFunc("/", withGzip(s.handleStatus))
	http.HandleFunc("/status", withGzip(s.handleStatus))
	http.HandleFunc("/health", s.handleHealth) // Health check usually small, no gzip needed
	http.HandleFunc("/diff", withGzip(s.handleDiff))
	http.HandleFunc("/diff/details", withGzip(s.handleDiffDetails))
	http.HandleFunc("/news", withGzip(s.handleNews))
	http.HandleFunc("/players", s.handlePlayers)
	http.HandleFunc("/depots", withGzip(s.handleDepots))
	http.HandleFunc("/servers", s.handleServers)

	http.HandleFunc("/steam", withGzip(s.handleStatus))
	http.HandleFunc("/steam/", withGzip(s.handleStatus))
	http.HandleFunc("/steam/status", withGzip(s.handleStatus))
	http.HandleFunc("/steam/health", s.handleHealth)
	http.HandleFunc("/steam/diff", withGzip(s.handleDiff))
	http.HandleFunc("/steam/diff/details", withGzip(s.handleDiffDetails))
	http.HandleFunc("/steam/news", withGzip(s.handleNews))
	http.HandleFunc("/steam/players", s.handlePlayers)
	http.HandleFunc("/steam/depots", withGzip(s.handleDepots))
	http.HandleFunc("/steam/servers", s.handleServers)

	log.Printf("API Server listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("API Server failed: %v", err)
	}
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	state := s.mon.GetState()
	uptime := time.Since(s.startTime)
	playerCount, _ := s.steamClient.GetPlayerCount(730)

	response := StatusResponse{
		AppID:         730,
		AppName:       "Counter-Strike 2",
		ChangeNumber:  state.ChangeNumber,
		BuildID:       state.BuildID,
		PlayerCount:   playerCount,
		Status:        "monitoring",
		UptimeSeconds: int64(uptime.Seconds()),
		HasUpdate:     state.LastDiff != nil,
		LastCheck:     time.Now().Unix(),
	}

	if state.LastDiff != nil {
		response.LastUpdate = &UpdateInfo{
			OldVersion:    state.LastDiff.OldVersion,
			NewVersion:    state.LastDiff.NewVersion,
			Type:          string(state.LastDiff.Type),
			TypeReason:    state.LastDiff.TypeReason,
			DepotsChanged: len(state.LastDiff.ChangedDepots),
			NewProtobufs:  len(state.LastDiff.NewProtobufs),
			NewStrings:    len(state.LastDiff.NewStrings),
		}
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
	})
}

func (s *Server) handleDiff(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	state := s.mon.GetState()

	if state.LastDiff == nil {
		json.NewEncoder(w).Encode(DiffResponse{HasDiff: false})
		return
	}

	var depots []DepotChangeAPI
	for _, d := range state.LastDiff.ChangedDepots {
		depots = append(depots, DepotChangeAPI{
			ID:     d.ID,
			Name:   d.Name,
			OldGID: d.OldGID,
			NewGID: d.NewGID,
		})
	}

	json.NewEncoder(w).Encode(DiffResponse{
		HasDiff:      true,
		OldVersion:   state.LastDiff.OldVersion,
		NewVersion:   state.LastDiff.NewVersion,
		Type:         string(state.LastDiff.Type),
		TypeReason:   state.LastDiff.TypeReason,
		Depots:       depots,
		NewProtobufs: state.LastDiff.NewProtobufs,
		NewStrings:   state.LastDiff.NewStrings,
		Analysis:     state.LastDiff.Analysis,
	})
}

func (s *Server) handleNews(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	news, err := s.steamClient.GetNews(730, 10)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var items []NewsItemAPI
	for _, n := range news {
		items = append(items, NewsItemAPI{
			Title:    n.Title,
			URL:      n.URL,
			Author:   n.Author,
			Contents: n.Contents,
			Date:     n.Date,
			Feed:     n.FeedLabel,
		})
	}

	json.NewEncoder(w).Encode(NewsResponse{
		AppID: 730,
		Count: len(items),
		News:  items,
	})
}

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	count, err := s.steamClient.GetPlayerCount(730)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(PlayersResponse{
		AppID:       730,
		PlayerCount: count,
		Timestamp:   time.Now().Unix(),
	})
}

func (s *Server) handleDepots(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	depots := []DepotInfo{
		{ID: "731", Name: "Public", Platform: "all", Type: "content"},
		{ID: "732", Name: "Public (Beta)", Platform: "all", Type: "content"},
		{ID: "733", Name: "Public (Debug)", Platform: "all", Type: "content"},
		{ID: "734", Name: "Binaries", Platform: "windows32", Type: "binary"},
		{ID: "735", Name: "Binaries Win64", Platform: "windows64", Type: "binary"},
		{ID: "736", Name: "Binaries Linux", Platform: "linux64", Type: "binary"},
		{ID: "737", Name: "Binaries Mac", Platform: "macos", Type: "binary"},
		{ID: "738", Name: "Binaries Mac ARM", Platform: "macos_arm", Type: "binary"},
		{ID: "2347770", Name: "CS2 Content", Platform: "all", Type: "content"},
		{ID: "2347771", Name: "CS2 Low Violence", Platform: "all", Type: "content"},
		{ID: "2347779", Name: "CS2 Dedicated Server", Platform: "all", Type: "server"},
	}

	state := s.mon.GetState()
	var changed []DepotChangeAPI
	if state.LastDiff != nil {
		for _, d := range state.LastDiff.ChangedDepots {
			changed = append(changed, DepotChangeAPI{
				ID:     d.ID,
				Name:   d.Name,
				OldGID: d.OldGID,
				NewGID: d.NewGID,
			})
		}
	}

	json.NewEncoder(w).Encode(DepotsResponse{
		AppID:       730,
		TotalDepots: len(depots),
		Depots:      depots,
		LastChanged: changed,
	})
}

type StatusResponse struct {
	AppID         int         `json:"app_id"`
	AppName       string      `json:"app_name"`
	ChangeNumber  string      `json:"change_number"`
	BuildID       string      `json:"build_id"`
	PlayerCount   int         `json:"player_count"`
	Status        string      `json:"status"`
	UptimeSeconds int64       `json:"uptime_seconds"`
	HasUpdate     bool        `json:"has_update"`
	LastCheck     int64       `json:"last_check"`
	LastUpdate    *UpdateInfo `json:"last_update,omitempty"`
}

type UpdateInfo struct {
	OldVersion    string `json:"old_version"`
	NewVersion    string `json:"new_version"`
	Type          string `json:"type"`
	TypeReason    string `json:"type_reason"`
	DepotsChanged int    `json:"depots_changed"`
	NewProtobufs  int    `json:"new_protobufs"`
	NewStrings    int    `json:"new_strings"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

type DiffResponse struct {
	HasDiff      bool             `json:"has_diff"`
	OldVersion   string           `json:"old_version,omitempty"`
	NewVersion   string           `json:"new_version,omitempty"`
	Type         string           `json:"type,omitempty"`
	TypeReason   string           `json:"type_reason,omitempty"`
	Depots       []DepotChangeAPI `json:"depots,omitempty"`
	NewProtobufs []string         `json:"new_protobufs,omitempty"`
	NewStrings   []string         `json:"new_strings,omitempty"`
	Analysis     string           `json:"analysis,omitempty"`
}

type DepotChangeAPI struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	OldGID string `json:"old_gid"`
	NewGID string `json:"new_gid"`
}

type NewsResponse struct {
	AppID int           `json:"app_id"`
	Count int           `json:"count"`
	News  []NewsItemAPI `json:"news"`
}

type NewsItemAPI struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Author   string `json:"author"`
	Contents string `json:"contents"`
	Date     int64  `json:"date"`
	Feed     string `json:"feed"`
}

type PlayersResponse struct {
	AppID       int   `json:"app_id"`
	PlayerCount int   `json:"player_count"`
	Timestamp   int64 `json:"timestamp"`
}

type DepotsResponse struct {
	AppID       int              `json:"app_id"`
	TotalDepots int              `json:"total_depots"`
	Depots      []DepotInfo      `json:"depots"`
	LastChanged []DepotChangeAPI `json:"last_changed"`
}

type DepotInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Platform string `json:"platform"`
	Type     string `json:"type"`
}

func (s *Server) handleServers(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	status, err := s.steamClient.GetServerStatus()
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(status)
}

type DiffDetailsResponse struct {
	HasData      bool            `json:"has_data"`
	OldVersion   string          `json:"old_version"`
	NewVersion   string          `json:"new_version"`
	Type         string          `json:"type"`
	TypeReason   string          `json:"type_reason"`
	Analysis     string          `json:"analysis"`
	StringBlocks []StringBlock   `json:"string_blocks"`
	ProtobufList []string        `json:"protobuf_list"`
	DepotBlocks  []DepotBlockAPI `json:"depot_blocks"`
	Timestamp    int64           `json:"timestamp"`
}

type StringBlock struct {
	Category string   `json:"category"`
	Icon     string   `json:"icon"`
	Count    int      `json:"count"`
	Strings  []string `json:"strings"`
}

type DepotBlockAPI struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	OldGID   string `json:"old_gid"`
	NewGID   string `json:"new_gid"`
	Platform string `json:"platform"`
}

func (s *Server) handleDiffDetails(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	state := s.mon.GetState()

	if state.LastDiff == nil {
		json.NewEncoder(w).Encode(DiffDetailsResponse{
			HasData:   false,
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// Cache check based on ChangeNumber
	etag := "W/" + "\"" + state.ChangeNumber + "\""
	if checkCache(w, r, etag) {
		return
	}
	diffData := state.LastDiff

	var stringBlocks []StringBlock

	// Use pre-computed categories if available
	var sourceBlocks []diff.CategoryBlock
	if len(diffData.CategorizedStrings) > 0 {
		sourceBlocks = diffData.CategorizedStrings
	} else {
		// Fallback for old data
		sourceBlocks = diff.CategorizeStrings(diffData.NewStrings)
	}

	for _, cat := range sourceBlocks {
		stringBlocks = append(stringBlocks, StringBlock{
			Category: cat.Category,
			Icon:     cat.Icon,
			Count:    cat.Count,
			Strings:  cat.Strings,
		})
	}

	depotBlocks := make([]DepotBlockAPI, 0, len(diffData.ChangedDepots))
	for _, d := range diffData.ChangedDepots {
		depotBlocks = append(depotBlocks, DepotBlockAPI{
			ID:       d.ID,
			Name:     d.Name,
			OldGID:   d.OldGID,
			NewGID:   d.NewGID,
			Platform: getDepotPlatform(d.ID),
		})
	}

	response := DiffDetailsResponse{
		HasData:      true,
		OldVersion:   diffData.OldVersion,
		NewVersion:   diffData.NewVersion,
		Type:         string(diffData.Type),
		TypeReason:   diffData.TypeReason,
		Analysis:     diffData.Analysis,
		StringBlocks: stringBlocks,
		ProtobufList: diffData.NewProtobufs,
		DepotBlocks:  depotBlocks,
		Timestamp:    time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(response)
}

func getDepotPlatform(depotID string) string {
	platforms := map[string]string{
		"731":     "Windows",
		"732":     "macOS",
		"733":     "Linux",
		"734":     "Dedicated Server",
		"2347771": "Windows (2347771)",
	}
	if p, ok := platforms[depotID]; ok {
		return p
	}
	return "Common"
}

// Middleware & Helpers

func withGzip(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCORS(w) // Ensure CORS is set before anything else
		if r.Method == "OPTIONS" {
			return
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()

		gzw := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next(&gzw, r)
	}
}

type gzipResponseWriter struct {
	*gzip.Writer
	ResponseWriter http.ResponseWriter
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func checkCache(w http.ResponseWriter, r *http.Request, etag string) bool {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=60") // Cache for 1 min, but revalidate with ETag

	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == etag {
			w.WriteHeader(http.StatusNotModified)
			return true
		}
	}
	return false
}
