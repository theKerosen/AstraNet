package steamcmd

import (
	"regexp"
	"strings"
)

type AppInfo struct {
	AppID        string                `json:"app_id"`
	Name         string                `json:"name"`
	ChangeNumber string                `json:"change_number"`
	BuildID      string                `json:"build_id"`
	Depots       map[string]DepotInfo  `json:"depots"`
	Branches     map[string]BranchInfo `json:"branches"`
	Common       map[string]string     `json:"common"`
	Config       map[string]string     `json:"config"`
}

type DepotInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	GID       string            `json:"gid"`
	Size      string            `json:"size"`
	MaxSize   string            `json:"maxsize"`
	Config    map[string]string `json:"config,omitempty"`
	Manifests map[string]string `json:"manifests,omitempty"`
}

type BranchInfo struct {
	Name        string `json:"name"`
	BuildID     string `json:"buildid"`
	TimeUpdated string `json:"timeupdated"`
	Description string `json:"description"`
	PWDRequired bool   `json:"pwdrequired"`
}

func ParseAppInfo(output string) *AppInfo {
	info := &AppInfo{
		Depots:   make(map[string]DepotInfo),
		Branches: make(map[string]BranchInfo),
		Common:   make(map[string]string),
		Config:   make(map[string]string),
	}

	statusLineRegex := regexp.MustCompile(`change number : (\d+)`)
	if matches := statusLineRegex.FindStringSubmatch(output); len(matches) > 1 {
		info.ChangeNumber = matches[1]
	}

	if info.ChangeNumber == "" {
		changeNumRegex := regexp.MustCompile(`(?i)"changenumber"\s+"?(\d+)"?`)
		if matches := changeNumRegex.FindStringSubmatch(output); len(matches) > 1 {
			info.ChangeNumber = matches[1]
		}
	}

	appIDStatusRegex := regexp.MustCompile(`AppID : (\d+)`)
	if matches := appIDStatusRegex.FindStringSubmatch(output); len(matches) > 1 {
		info.AppID = matches[1]
	}

	if info.AppID == "" {
		appIDRegex := regexp.MustCompile(`"appid"\s+"?(\d+)"?`)
		if matches := appIDRegex.FindStringSubmatch(output); len(matches) > 1 {
			info.AppID = matches[1]
		}
	}

	nameRegex := regexp.MustCompile(`"name"\s+"([^"]+)"`)
	if matches := nameRegex.FindStringSubmatch(output); len(matches) > 1 {
		info.Name = matches[1]
	}

	buildIDRegex := regexp.MustCompile(`"buildid"\s+"(\d+)"`)
	if matches := buildIDRegex.FindStringSubmatch(output); len(matches) > 1 {
		info.BuildID = matches[1]
	}

	parseCommon(output, info)
	parseDepots(output, info)
	parseBranches(output, info)

	return info
}

func parseCommon(output string, info *AppInfo) {
	commonFields := []string{"type", "oslist", "clienticon", "clienttga", "icon", "logo", "logo_small", "controller_support"}

	for _, field := range commonFields {
		regex := regexp.MustCompile(`"` + field + `"\s+"([^"]+)"`)
		if matches := regex.FindStringSubmatch(output); len(matches) > 1 {
			info.Common[field] = matches[1]
		}
	}
}

func parseDepots(output string, info *AppInfo) {
	lines := strings.Split(output, "\n")
	var currentDepotID string
	var inManifests bool
	var inPublic bool
	var currentBranch string
	depth := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "{" {
			depth++
			continue
		}
		if line == "}" {
			depth--
			if depth < 3 {
				currentDepotID = ""
				inManifests = false
				inPublic = false
			}
			continue
		}

		if match, _ := regexp.MatchString(`^"\d+"$`, line); match {
			potentialID := strings.Trim(line, "\"")
			if len(potentialID) >= 3 && len(potentialID) <= 10 {
				currentDepotID = potentialID
				if _, exists := info.Depots[currentDepotID]; !exists {
					info.Depots[currentDepotID] = DepotInfo{
						ID:        currentDepotID,
						Manifests: make(map[string]string),
						Config:    make(map[string]string),
					}
				}
				inManifests = false
				inPublic = false
			}
			continue
		}

		if line == `"manifests"` {
			inManifests = true
			continue
		}

		if inManifests {
			branchMatch := regexp.MustCompile(`^"([^"]+)"$`).FindStringSubmatch(line)
			if len(branchMatch) > 1 {
				currentBranch = branchMatch[1]
				inPublic = true
				continue
			}
		}

		if inPublic && strings.HasPrefix(line, `"gid"`) {
			parts := strings.Fields(line)
			if len(parts) >= 2 && currentDepotID != "" {
				gid := strings.Trim(parts[1], "\"")
				depot := info.Depots[currentDepotID]
				depot.Manifests[currentBranch] = gid
				if currentBranch == "public" {
					depot.GID = gid
				}
				info.Depots[currentDepotID] = depot
			}
		}

		if currentDepotID != "" {
			if strings.HasPrefix(line, `"maxsize"`) {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					depot := info.Depots[currentDepotID]
					depot.MaxSize = strings.Trim(parts[1], "\"")
					info.Depots[currentDepotID] = depot
				}
			}
			if strings.HasPrefix(line, `"name"`) && currentDepotID != "" {
				parts := strings.SplitN(line, "\"", 4)
				if len(parts) >= 4 {
					depot := info.Depots[currentDepotID]
					depot.Name = parts[3]
					info.Depots[currentDepotID] = depot
				}
			}
		}
	}
}

func parseBranches(output string, info *AppInfo) {
	branchRegex := regexp.MustCompile(`"(\w+)"\s*\{\s*"buildid"\s+"(\d+)"`)
	matches := branchRegex.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			branchName := match[1]
			buildID := match[2]

			if branchName != "public" && branchName != "beta" && branchName != "preview" {
				continue
			}

			info.Branches[branchName] = BranchInfo{
				Name:    branchName,
				BuildID: buildID,
			}
		}
	}

	timeRegex := regexp.MustCompile(`"timeupdated"\s+"(\d+)"`)
	if matches := timeRegex.FindAllStringSubmatch(output, 1); len(matches) > 0 {
		if branch, exists := info.Branches["public"]; exists {
			branch.TimeUpdated = matches[0][1]
			info.Branches["public"] = branch
		}
	}
}

func IsAppInfoOutput(line string) bool {
	return strings.TrimSpace(line) == "}"
}
