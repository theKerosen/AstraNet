package notifier

import (
	"astra_core/database"
	"astra_core/diff"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DiscordNotifier struct {
	db *database.DB
}

func NewDiscordNotifier(db *database.DB) *DiscordNotifier {
	return &DiscordNotifier{db: db}
}

type WebhookPayload struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Thumbnail   *EmbedImage  `json:"thumbnail,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
	URL         string       `json:"url,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedImage struct {
	URL string `json:"url"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// StatusUpdate represents a change in service status
type StatusUpdate struct {
	Service       string // "Steam" or "CS2"
	OldStatus     string
	NewStatus     string
	IsMaintenance bool
}

func (n *DiscordNotifier) broadcast(payload WebhookPayload, files map[string][]byte) error {
	urls, err := n.db.GetAllWebhooks()
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			if err := n.send(u, payload, files); err != nil {
				log.Printf("Failed to send webhook to %s: %v", u, err)
			}
		}(url)
	}
	wg.Wait()
	return nil
}

func (n *DiscordNotifier) send(url string, payload WebhookPayload, files map[string][]byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for filename, content := range files {
		part, err := writer.CreateFormFile("files["+filename+"]", filename)
		if err != nil {
			return err
		}
		part.Write(content)
	}

	if err := writer.WriteField("payload_json", string(payloadBytes)); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	resp, err := http.Post(url, writer.FormDataContentType(), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	return nil
}

func (n *DiscordNotifier) NotifyStatus(update StatusUpdate) error {
	color := 0x00FF00 // Green
	title := fmt.Sprintf("Serviços Online: %s", update.Service)
	description := "O serviço está operando normalmente."

	if update.NewStatus == "offline" || update.NewStatus == "critical" {
		if update.IsMaintenance {
			color = 0xFFA500 // Orange
			title = "Manutenção Steam"
			description = "Manutenção de rotina detectada. Serviços podem estar instáveis."
		} else {
			color = 0xFF0000 // Red
			title = fmt.Sprintf("Alerta de Serviço: %s", update.Service)
			description = fmt.Sprintf("O serviço está atualmente **%s**.", strings.ToUpper(update.NewStatus))
		}
	} else if update.NewStatus == "online" && update.OldStatus != "online" {
		title = fmt.Sprintf("Serviço Recuperado: %s", update.Service)
		description = "O serviço voltou a operar normalmente."
	}

	embed := Embed{
		Title:       title,
		Description: description,
		Color:       color,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &EmbedFooter{
			Text: "AstraNet • https://ladyluh.dev",
		},
	}

	return n.broadcast(WebhookPayload{Embeds: []Embed{embed}}, nil)
}

func (n *DiscordNotifier) Notify(result *diff.DiffResult) error {
	color := getColorForUpdateType(result.Type)

	embed := Embed{
		Title:       "Counter-Strike 2 — Update Detected",
		Description: fmt.Sprintf("~~*%s*~~ → `%s`", result.OldVersion, result.NewVersion),
		Color:       color,
		Thumbnail: &EmbedImage{
			URL: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg",
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &EmbedFooter{
			Text: "AstraNet • https://ladyluh.dev",
		},
	}

	if result.Type != diff.UpdateTypeUnknown {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Update Type",
			Value:  fmt.Sprintf("**%s**", result.Type),
			Inline: true,
		})
	}

	if result.TypeReason != "" {
		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Reason",
			Value:  result.TypeReason,
			Inline: true,
		})
	}

	if len(result.ChangedDepots) > 0 {
		var content strings.Builder
		for i, depot := range result.ChangedDepots {
			if i >= 5 {
				content.WriteString(fmt.Sprintf("... and %d more\n", len(result.ChangedDepots)-5))
				break
			}
			name := depot.Name
			if name == "" {
				name = "Unknown Depot"
			}
			content.WriteString(fmt.Sprintf("**%s** (`%s`)\n", name, depot.ID))
		}

		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Changed Depots",
			Value:  content.String(),
			Inline: false,
		})
	}

	if len(result.StringBlocks) > 0 {
		var notable strings.Builder
		count := 0
		for _, block := range result.StringBlocks {
			for _, s := range block.Strings {
				if count >= 10 {
					break
				}
				if len(s) < 50 { // simple filter
					notable.WriteString(fmt.Sprintf("`%s`\n", s))
					count++
				}
			}
			if count >= 10 {
				notable.WriteString("... and more")
				break
			}
		}

		if notable.Len() > 0 {
			embed.Fields = append(embed.Fields, EmbedField{
				Name:   "Notable Strings",
				Value:  notable.String(),
				Inline: false,
			})
		}
	}

	files := make(map[string][]byte)
	if result.RawDiff != "" {
		files["vdf_diff.txt"] = []byte(result.RawDiff)
	}
	if result.Analysis != "" {
		files["analysis.md"] = []byte(result.Analysis)
	}

	return n.broadcast(WebhookPayload{Embeds: []Embed{embed}}, files)
}

func getColorForUpdateType(t diff.UpdateType) int {
	colors := map[diff.UpdateType]int{
		diff.UpdateTypeUnknown:      0x808080,
		diff.UpdateTypeFeature:      0x00FF00,
		diff.UpdateTypePatch:        0x00BFFF,
		diff.UpdateTypeMap:          0xFFD700,
		diff.UpdateTypeItem:         0xFF69B4,
		diff.UpdateTypeLocalization: 0x9370DB,
		diff.UpdateTypeServer:       0xFF4500,
		diff.UpdateTypeBalance:      0xFFA500,
		diff.UpdateTypeAntiCheat:    0xFF0000,
		diff.UpdateTypeCosmetic:     0xFF1493,
		diff.UpdateTypeProtobuf:     0x7B68EE,
	}

	if color, ok := colors[t]; ok {
		return color
	}
	return 0x00FF00
}
