package diff

import (
	"astra_core/steamcmd"
	"strings"
)

type DiffResult struct {
	NewVersion       string        `json:"new_version"`
	OldVersion       string        `json:"old_version"`
	ChangedFiles     []string      `json:"changed_files"`
	NewFiles         []string      `json:"new_files"`
	RemovedFiles     []string      `json:"removed_files"`
	ChangedDepots    []DepotChange `json:"changed_depots"`
	RawDiff          string        `json:"raw_diff,omitempty"`
	Type             UpdateType    `json:"type"`
	TypeReason       string        `json:"type_reason,omitempty"`
	NewProtobufs     []string      `json:"new_protobufs,omitempty"`
	RemovedProtobufs []string      `json:"removed_protobufs,omitempty"`
	NewStrings       []string      `json:"new_strings,omitempty"`
	Analysis         string        `json:"analysis,omitempty"`
}

type DepotChange struct {
	ID     string `json:"id"`
	OldGID string `json:"old_gid"`
	NewGID string `json:"new_gid"`
	Name   string `json:"name"`
}

type UpdateType string

const (
	UpdateTypeUnknown      UpdateType = "Unknown"
	UpdateTypeFeature      UpdateType = "Feature"
	UpdateTypePatch        UpdateType = "Patch"
	UpdateTypeMap          UpdateType = "Map"
	UpdateTypeItem         UpdateType = "Item"
	UpdateTypeLocalization UpdateType = "Localization"
	UpdateTypeServer       UpdateType = "Server"
	UpdateTypeBalance      UpdateType = "Balance"
	UpdateTypeAntiCheat    UpdateType = "Anti-Cheat"
	UpdateTypeCosmetic     UpdateType = "Cosmetic"
	UpdateTypeProtobuf     UpdateType = "Protobuf/Networking"
)

type Tracker struct {
	client *steamcmd.Client
}

func NewTracker(client *steamcmd.Client) *Tracker {
	return &Tracker{client: client}
}

func (t *Tracker) ProcessUpdate(oldInfo, newInfo *steamcmd.AppInfo) *DiffResult {
	result := &DiffResult{
		NewVersion: newInfo.ChangeNumber,
		OldVersion: oldInfo.ChangeNumber,
		Type:       UpdateTypeUnknown,
	}

	for depotID, newDepot := range newInfo.Depots {
		oldDepot, exists := oldInfo.Depots[depotID]
		if !exists {
			result.ChangedDepots = append(result.ChangedDepots, DepotChange{
				ID:     depotID,
				OldGID: "",
				NewGID: newDepot.GID,
				Name:   getDepotName(depotID),
			})
			continue
		}

		if newDepot.GID != oldDepot.GID {
			result.ChangedDepots = append(result.ChangedDepots, DepotChange{
				ID:     depotID,
				OldGID: oldDepot.GID,
				NewGID: newDepot.GID,
				Name:   getDepotName(depotID),
			})
		}
	}

	result.Type, result.TypeReason = classifyUpdateByDepots(result.ChangedDepots)

	return result
}

func (t *Tracker) EnhanceWithStringAnalysis(result *DiffResult, newStrings, oldStrings []string) {
	added, removed := compareStrings(oldStrings, newStrings)

	result.NewStrings = filterTopStrings(added, 50)

	newType, reason := classifyByStrings(added, removed)
	if newType != UpdateTypeUnknown {
		result.Type = newType
		result.TypeReason = reason
	}

	result.Analysis = generateAnalysis(added, removed, result.Type)
}

func classifyUpdateByDepots(depots []DepotChange) (UpdateType, string) {
	for _, depot := range depots {
		switch depot.ID {
		case "2347779":
			return UpdateTypeServer, "CS2 Dedicated Server depot changed"
		case "731":
			return UpdateTypePatch, "Public depot changed"
		case "2347770":
			return UpdateTypePatch, "CS2 Content depot changed"
		}
	}
	return UpdateTypeUnknown, ""
}

func classifyByStrings(added, removed []string) (UpdateType, string) {
	var weaponCount, protoCount, balanceCount, cosmeticCount int

	for _, s := range added {
		lower := strings.ToLower(s)

		if strings.HasPrefix(lower, "cmsg") || strings.Contains(lower, "proto") {
			protoCount++
		}
		if strings.HasPrefix(lower, "weapon_") {
			weaponCount++
		}
		if strings.HasPrefix(lower, "item_") || strings.Contains(lower, "cosmetic") {
			cosmeticCount++
		}
		if strings.Contains(lower, "damage") || strings.Contains(lower, "armor") ||
			strings.Contains(lower, "speed") || strings.Contains(lower, "accuracy") {
			balanceCount++
		}
	}

	if protoCount > 5 {
		return UpdateTypeProtobuf, "Multiple new protobuf definitions detected"
	}
	if weaponCount > 3 {
		return UpdateTypeBalance, "Multiple weapon-related strings detected"
	}
	if cosmeticCount > 5 {
		return UpdateTypeCosmetic, "Multiple cosmetic-related strings detected"
	}
	if balanceCount > 5 {
		return UpdateTypeBalance, "Multiple balance-related strings detected"
	}

	return UpdateTypeUnknown, ""
}

func compareStrings(old, new []string) (added, removed []string) {
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)

	for _, s := range old {
		oldSet[s] = true
	}
	for _, s := range new {
		newSet[s] = true
	}

	for s := range newSet {
		if !oldSet[s] {
			added = append(added, s)
		}
	}
	for s := range oldSet {
		if !newSet[s] {
			removed = append(removed, s)
		}
	}

	return
}

func filterTopStrings(strings []string, limit int) []string {
	if len(strings) <= limit {
		return strings
	}
	return strings[:limit]
}

func generateAnalysis(added, removed []string, updateType UpdateType) string {
	var sb strings.Builder

	sb.WriteString("## Update Analysis\n\n")
	sb.WriteString("**Detected Type:** " + string(updateType) + "\n\n")

	if len(added) > 0 {
		sb.WriteString("**Notable Additions:**\n")
		count := 0
		for _, s := range added {
			if isNotableString(s) && count < 20 {
				sb.WriteString("- `" + s + "`\n")
				count++
			}
		}
	}

	return sb.String()
}

func isNotableString(s string) bool {
	lower := strings.ToLower(s)
	notable := []string{"weapon_", "item_", "cmsg", "ability_", "hero_", "npc_", "k_e", "sv_", "mp_", "cl_"}

	for _, prefix := range notable {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func getDepotName(depotID string) string {
	names := map[string]string{
		"731":     "Public",
		"732":     "Public (Beta)",
		"733":     "Public (Debug)",
		"734":     "Binaries",
		"735":     "Binaries Win64",
		"736":     "Binaries Linux",
		"737":     "Binaries Mac",
		"738":     "Binaries Mac ARM",
		"2347770": "CS2 Content",
		"2347771": "CS2 Content (Low Violence)",
		"2347772": "CS2 Content Asia",
		"2347773": "CS2 Workshop",
		"2347774": "CS2 Workshop Linux",
		"2347779": "CS2 Dedicated Server",
	}

	if name, ok := names[depotID]; ok {
		return name
	}
	return "Unknown Depot"
}
