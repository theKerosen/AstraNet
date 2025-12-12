package extractor

import (
	"regexp"
	"strings"
)

var protobufPatterns = []*regexp.Regexp{
	regexp.MustCompile(`CMsg[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CUser[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CClient[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CServer[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CMsgGC[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CMsgDOTA[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`CMsgCS[A-Z][a-zA-Z0-9_]+`),
	regexp.MustCompile(`k_E[A-Z][a-zA-Z0-9_]+`),
}

type ProtobufMatch struct {
	Name    string
	Type    string
	IsNew   bool
	Context string
}

func ExtractProtobufs(strings []string) []ProtobufMatch {
	var matches []ProtobufMatch
	seen := make(map[string]bool)

	for _, s := range strings {
		for _, pattern := range protobufPatterns {
			found := pattern.FindAllString(s, -1)
			for _, match := range found {
				if seen[match] {
					continue
				}
				seen[match] = true

				matches = append(matches, ProtobufMatch{
					Name:    match,
					Type:    classifyProtobuf(match),
					Context: s,
				})
			}
		}
	}

	return matches
}

func classifyProtobuf(name string) string {
	lower := strings.ToLower(name)

	switch {
	case strings.Contains(lower, "gc"):
		return "GameCoordinator"
	case strings.Contains(lower, "client"):
		return "Client"
	case strings.Contains(lower, "server"):
		return "Server"
	case strings.Contains(lower, "user"):
		return "User"
	case strings.HasPrefix(lower, "k_e"):
		return "Enum"
	case strings.Contains(lower, "dota"):
		return "DOTA2"
	case strings.Contains(lower, "cs"):
		return "CS2"
	default:
		return "Message"
	}
}

func CompareProtobufs(oldProtos, newProtos []ProtobufMatch) (added, removed []ProtobufMatch) {
	oldSet := make(map[string]ProtobufMatch)
	newSet := make(map[string]ProtobufMatch)

	for _, p := range oldProtos {
		oldSet[p.Name] = p
	}
	for _, p := range newProtos {
		newSet[p.Name] = p
	}

	for name, proto := range newSet {
		if _, exists := oldSet[name]; !exists {
			proto.IsNew = true
			added = append(added, proto)
		}
	}

	for name, proto := range oldSet {
		if _, exists := newSet[name]; !exists {
			removed = append(removed, proto)
		}
	}

	return added, removed
}

func AnalyzeProtobufChanges(added, removed []ProtobufMatch) string {
	if len(added) == 0 && len(removed) == 0 {
		return ""
	}

	var sb strings.Builder

	if len(added) > 0 {
		sb.WriteString("**New Protobufs:**\n")
		for _, p := range added {
			sb.WriteString("+ `" + p.Name + "` (" + p.Type + ")\n")
		}
	}

	if len(removed) > 0 {
		sb.WriteString("\n**Removed Protobufs:**\n")
		for _, p := range removed {
			sb.WriteString("- `" + p.Name + "` (" + p.Type + ")\n")
		}
	}

	return sb.String()
}
