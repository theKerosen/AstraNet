package notifier

import (
	"astra_core/diff"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type DiscordNotifier struct {
	WebhookURL string
}

func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{WebhookURL: webhookURL}
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
	Text string `json:"text"`
}

func (n *DiscordNotifier) Notify(result *diff.DiffResult) error {
	if n.WebhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	color := getColorForUpdateType(result.Type)

	embed := Embed{
		Title:       "Counter-Strike 2 — Update Detected",
		Description: fmt.Sprintf("~~*%s*~~ → `%s`", result.OldVersion, result.NewVersion),
		Color:       color,
		Thumbnail: &EmbedImage{
			URL: "https://cdn.cloudflare.steamstatic.com/steamcommunity/public/images/apps/730/8dbc71957312bbd3baea65848b545be9eae2a355.jpg",
		},
		Timestamp: time.Now().Format(time.RFC3339),
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
			oldGid := depot.OldGID
			if oldGid == "" {
				oldGid = "New"
			}
			content.WriteString(fmt.Sprintf("**%s** (`%s`)\n", name, depot.ID))
		}

		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Changed Depots",
			Value:  content.String(),
			Inline: false,
		})
	}

	if len(result.NewProtobufs) > 0 {
		var protoList strings.Builder
		for i, proto := range result.NewProtobufs {
			if i >= 10 {
				protoList.WriteString(fmt.Sprintf("... and %d more", len(result.NewProtobufs)-10))
				break
			}
			protoList.WriteString(fmt.Sprintf("`%s`\n", proto))
		}

		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "New Protobufs",
			Value:  protoList.String(),
			Inline: false,
		})
	}

	if len(result.NewStrings) > 0 {
		var stringList strings.Builder
		for i, s := range result.NewStrings {
			if i >= 10 {
				stringList.WriteString(fmt.Sprintf("... and %d more", len(result.NewStrings)-10))
				break
			}
			stringList.WriteString(fmt.Sprintf("`%s`\n", s))
		}

		embed.Fields = append(embed.Fields, EmbedField{
			Name:   "Notable Strings",
			Value:  stringList.String(),
			Inline: false,
		})
	}

	embed.Footer = &EmbedFooter{
		Text: "AstraNet • Steam Update Monitor",
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	payload := WebhookPayload{
		Embeds: []Embed{embed},
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if result.RawDiff != "" {
		part, err := writer.CreateFormFile("files[0]", "vdf_diff.txt")
		if err != nil {
			return err
		}
		part.Write([]byte(result.RawDiff))
	}

	if result.Analysis != "" {
		part, err := writer.CreateFormFile("files[1]", "analysis.md")
		if err != nil {
			return err
		}
		part.Write([]byte(result.Analysis))
	}

	if err := writer.WriteField("payload_json", string(payloadBytes)); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	resp, err := http.Post(n.WebhookURL, writer.FormDataContentType(), body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord webhook failed with status: %d", resp.StatusCode)
	}

	log.Println("Discord notification sent successfully.")
	return nil
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
