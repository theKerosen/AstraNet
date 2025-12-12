package steam

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	SteamAPIBaseURL = "https://api.steampowered.com"
	SteamStoreAPI   = "https://store.steampowered.com/api"
	CacheDuration   = 2 * time.Minute
)

type SteamWebClient struct {
	httpClient        *http.Client
	apiKey            string
	serverStatusCache *ServerStatus
	cacheTime         time.Time
	cacheMutex        sync.RWMutex
	playerCountCache  int
	playerCacheTime   time.Time
}

func NewSteamWebClient() *SteamWebClient {
	return &SteamWebClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiKey:     os.Getenv("STEAM_API_KEY"),
	}
}

type NewsResponse struct {
	AppNews struct {
		AppID     int        `json:"appid"`
		NewsItems []NewsItem `json:"newsitems"`
	} `json:"appnews"`
}

type NewsItem struct {
	GID        string `json:"gid"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	Author     string `json:"author"`
	Contents   string `json:"contents"`
	FeedLabel  string `json:"feedlabel"`
	Date       int64  `json:"date"`
	FeedName   string `json:"feedname"`
	FeedType   int    `json:"feed_type"`
	AppID      int    `json:"appid"`
	IsExternal bool   `json:"is_external_url"`
}

type AppDetails struct {
	Success bool           `json:"success"`
	Data    AppDetailsData `json:"data"`
}

type AppDetailsData struct {
	Type                string   `json:"type"`
	Name                string   `json:"name"`
	AppID               int      `json:"steam_appid"`
	RequiredAge         int      `json:"required_age"`
	IsFree              bool     `json:"is_free"`
	DetailedDescription string   `json:"detailed_description"`
	AboutTheGame        string   `json:"about_the_game"`
	ShortDescription    string   `json:"short_description"`
	SupportedLanguages  string   `json:"supported_languages"`
	HeaderImage         string   `json:"header_image"`
	Website             string   `json:"website"`
	Developers          []string `json:"developers"`
	Publishers          []string `json:"publishers"`
	ReleaseDate         struct {
		ComingSoon bool   `json:"coming_soon"`
		Date       string `json:"date"`
	} `json:"release_date"`
	Metacritic struct {
		Score int    `json:"score"`
		URL   string `json:"url"`
	} `json:"metacritic"`
}

type PlayerCount struct {
	Response struct {
		PlayerCount int `json:"player_count"`
		Result      int `json:"result"`
	} `json:"response"`
}

type ServerStatus struct {
	Steam       string `json:"steam"`
	CS2         string `json:"cs2"`
	Matchmaking string `json:"matchmaking"`
	Sessions    string `json:"sessions"`
	Scheduler   string `json:"scheduler"`
	OnlineCount int    `json:"online_count"`
	Timestamp   int64  `json:"timestamp"`
	Cached      bool   `json:"cached"`
}

type CSGOServerStatusResponse struct {
	Result struct {
		App struct {
			Version   int    `json:"version"`
			Timestamp int64  `json:"timestamp"`
			Time      string `json:"time"`
		} `json:"app"`
		Services struct {
			SessionsLogon  string `json:"SessionsLogon"`
			SteamCommunity string `json:"SteamCommunity"`
			IEconItems     string `json:"IEconItems"`
			Leaderboards   string `json:"Leaderboards"`
		} `json:"services"`
		Datacenters map[string]struct {
			Capacity string `json:"capacity"`
			Load     string `json:"load"`
		} `json:"datacenters"`
		Matchmaking struct {
			Scheduler        string `json:"scheduler"`
			OnlineServers    int    `json:"online_servers"`
			OnlinePlayers    int    `json:"online_players"`
			SearchSecondsAvg int    `json:"searching_players"`
		} `json:"matchmaking"`
	} `json:"result"`
}

func (c *SteamWebClient) GetNews(appID int, count int) ([]NewsItem, error) {
	url := fmt.Sprintf("%s/ISteamNews/GetNewsForApp/v0002/?appid=%d&count=%d&maxlength=500&format=json", SteamAPIBaseURL, appID, count)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var newsResp NewsResponse
	if err := json.NewDecoder(resp.Body).Decode(&newsResp); err != nil {
		return nil, err
	}

	return newsResp.AppNews.NewsItems, nil
}

func (c *SteamWebClient) GetAppDetails(appID int) (*AppDetailsData, error) {
	url := fmt.Sprintf("%s/appdetails?appids=%d", SteamStoreAPI, appID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]AppDetails
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	appKey := fmt.Sprintf("%d", appID)
	if details, ok := result[appKey]; ok && details.Success {
		return &details.Data, nil
	}

	return nil, fmt.Errorf("app %d not found", appID)
}

func (c *SteamWebClient) GetPlayerCount(appID int) (int, error) {
	c.cacheMutex.RLock()
	if time.Since(c.playerCacheTime) < CacheDuration && c.playerCountCache > 0 {
		count := c.playerCountCache
		c.cacheMutex.RUnlock()
		return count, nil
	}
	c.cacheMutex.RUnlock()

	url := fmt.Sprintf("%s/ISteamUserStats/GetNumberOfCurrentPlayers/v1/?appid=%d", SteamAPIBaseURL, appID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result PlayerCount
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if result.Response.Result != 1 {
		return 0, fmt.Errorf("failed to get player count")
	}

	c.cacheMutex.Lock()
	c.playerCountCache = result.Response.PlayerCount
	c.playerCacheTime = time.Now()
	c.cacheMutex.Unlock()

	return result.Response.PlayerCount, nil
}

func (c *SteamWebClient) GetServerStatus() (*ServerStatus, error) {
	c.cacheMutex.RLock()
	if c.serverStatusCache != nil && time.Since(c.cacheTime) < CacheDuration {
		cached := *c.serverStatusCache
		cached.Cached = true
		c.cacheMutex.RUnlock()
		return &cached, nil
	}
	c.cacheMutex.RUnlock()

	status := &ServerStatus{
		Steam:       "unknown",
		CS2:         "unknown",
		Matchmaking: "unknown",
		Sessions:    "unknown",
		Scheduler:   "unknown",
		Timestamp:   time.Now().Unix(),
		Cached:      false,
	}

	playerCount, err := c.GetPlayerCount(730)
	if err == nil && playerCount > 0 {
		status.Steam = "online"
		status.CS2 = "online"
		status.OnlineCount = playerCount
	}

	if c.apiKey == "" {
		if playerCount > 100000 {
			status.Matchmaking = "normal"
		} else if playerCount > 0 {
			status.Matchmaking = "low"
		}
		c.cacheStatus(status)
		return status, nil
	}

	url := fmt.Sprintf("%s/ICSGOServers_730/GetGameServersStatus/v1/?key=%s", SteamAPIBaseURL, c.apiKey)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.cacheStatus(status)
		return status, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.cacheStatus(status)
		return status, nil
	}

	var csgoStatus CSGOServerStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&csgoStatus); err != nil {
		c.cacheStatus(status)
		return status, nil
	}

	if csgoStatus.Result.Services.SessionsLogon == "normal" {
		status.Sessions = "normal"
	} else {
		status.Sessions = csgoStatus.Result.Services.SessionsLogon
	}

	if csgoStatus.Result.Matchmaking.Scheduler == "normal" {
		status.Scheduler = "normal"
		status.Matchmaking = "normal"
	} else {
		status.Scheduler = csgoStatus.Result.Matchmaking.Scheduler
		status.Matchmaking = csgoStatus.Result.Matchmaking.Scheduler
	}

	if csgoStatus.Result.Matchmaking.OnlinePlayers > 0 {
		status.OnlineCount = csgoStatus.Result.Matchmaking.OnlinePlayers
	}

	c.cacheStatus(status)
	return status, nil
}

func (c *SteamWebClient) cacheStatus(status *ServerStatus) {
	c.cacheMutex.Lock()
	c.serverStatusCache = status
	c.cacheTime = time.Now()
	c.cacheMutex.Unlock()
}
