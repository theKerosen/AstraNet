package extractor

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"
)

const (
	MinStringLength = 4
	bufferSize      = 64 * 1024 // 64KB buffer for reading
)

var interestingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)CMsg[A-Z]`),
	regexp.MustCompile(`(?i)CUser`),
	regexp.MustCompile(`(?i)CClient`),
	regexp.MustCompile(`(?i)CServer`),
	regexp.MustCompile(`(?i)weapon_`),
	regexp.MustCompile(`(?i)item_`),
	regexp.MustCompile(`(?i)ability_`),
	regexp.MustCompile(`(?i)hero_`),
	regexp.MustCompile(`(?i)npc_`),
	regexp.MustCompile(`(?i)proto`),
	regexp.MustCompile(`(?i)_proto$`),
	regexp.MustCompile(`(?i)k_E[A-Z]`),
	regexp.MustCompile(`(?i)DOTA_`),
	regexp.MustCompile(`(?i)CS_`),
	regexp.MustCompile(`(?i)game\.`),
	regexp.MustCompile(`(?i)convar`),
	regexp.MustCompile(`(?i)cvar`),
	regexp.MustCompile(`(?i)sv_`),
	regexp.MustCompile(`(?i)mp_`),
	regexp.MustCompile(`(?i)cl_`),
	regexp.MustCompile(`(?i)de_`),
	regexp.MustCompile(`(?i)cs_`),
	regexp.MustCompile(`(?i)ar_`),
	regexp.MustCompile(`(?i)sf_ui_`),
	regexp.MustCompile(`(?i)hud_`),
	regexp.MustCompile(`(?i)panorama`),
	regexp.MustCompile(`(?i)sound`),
}

type StringMatch struct {
	Value    string
	Category string
}

// ExtractAndFilterStrings streams the file and filters strings on the fly to reduce memory usage
func ExtractAndFilterStrings(filePath string) ([]StringMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []StringMatch
	seen := make(map[string]bool)
	reader := bufio.NewReaderSize(file, bufferSize)

	var current []rune

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			break
		}

		if isPrintableRune(r) {
			current = append(current, r)
		} else {
			if len(current) >= MinStringLength {
				s := string(current)
				if !seen[s] {
					match, ok := evaluateString(s)
					if ok {
						matches = append(matches, match)
						seen[s] = true
					} else if isReasonableString(s) {
						// Keep "reasonable" strings but maybe as 'other' to not lose data,
						// but effectively we might want to be stricter here if perf is still bad.
						// For now, let's keep the logic similar but streaming.
						// Optimize: Don't store "other" if it's too generic?
						// Let's stick to the original logic for "other" but applied here.
						matches = append(matches, StringMatch{Value: s, Category: "other"})
						seen[s] = true
					}
				}
			}
			current = nil
		}
	}

	// Handle last buffer
	if len(current) >= MinStringLength {
		s := string(current)
		if !seen[s] {
			if match, ok := evaluateString(s); ok {
				matches = append(matches, match)
			} else if isReasonableString(s) {
				matches = append(matches, StringMatch{Value: s, Category: "other"})
			}
		}
	}

	return matches, nil
}

func evaluateString(s string) (StringMatch, bool) {
	for _, pattern := range interestingPatterns {
		if pattern.MatchString(s) {
			return StringMatch{
				Value:    s,
				Category: categorizeString(s),
			}, true
		}
	}
	return StringMatch{}, false
}

func isPrintableRune(r rune) bool {
	return unicode.IsPrint(r) && r < 128
}

// Deprecated: Use ExtractAndFilterStrings instead
func ExtractStrings(filePath string) ([]string, error) {
	// Implementation preserved for compatibility but inefficient
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return extractPrintableStrings(data), nil
}

func extractPrintableStrings(data []byte) []string {
	var strings []string
	var current []byte
	for _, b := range data {
		if isPrintable(b) {
			current = append(current, b)
		} else {
			if len(current) >= MinStringLength {
				strings = append(strings, string(current))
			}
			current = nil
		}
	}
	if len(current) >= MinStringLength {
		strings = append(strings, string(current))
	}
	return strings
}

func isPrintable(b byte) bool {
	r := rune(b)
	return unicode.IsPrint(r) && r < 128
}

func FilterInterestingStrings(strings []string) []StringMatch {
	// Compatibility wrapper
	var matches []StringMatch
	seen := make(map[string]bool)
	for _, s := range strings {
		if seen[s] {
			continue
		}
		if match, ok := evaluateString(s); ok {
			matches = append(matches, match)
			seen[s] = true
		} else if isReasonableString(s) {
			matches = append(matches, StringMatch{Value: s, Category: "other"})
			seen[s] = true
		}
	}
	return matches
}

func isReasonableString(s string) bool {
	if len(s) > 100 {
		return false
	} // Optimization: Ignore super long garbage strings
	hasLetter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
			break
		}
	}
	return hasLetter
}

func categorizeString(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "cmsg") || strings.Contains(lower, "proto"):
		return "protobuf"
	case strings.Contains(lower, "weapon_"):
		return "weapon"
	case strings.Contains(lower, "item_"):
		return "item"
	case strings.Contains(lower, "de_") || strings.Contains(lower, "cs_") || strings.Contains(lower, "ar_"):
		return "map"
	case strings.Contains(lower, "sf_ui_") || strings.Contains(lower, "hud_") || strings.Contains(lower, "panorama"):
		return "ui"
	case strings.Contains(lower, "sound") || strings.Contains(lower, "music") || strings.Contains(lower, "audio"):
		return "sound"
	case strings.Contains(lower, "ability_") || strings.Contains(lower, "hero_"):
		return "gameplay"
	case strings.Contains(lower, "npc_"):
		return "npc"
	case strings.Contains(lower, "sv_") || strings.Contains(lower, "mp_") || strings.Contains(lower, "cl_"):
		return "convar"
	case strings.Contains(lower, "k_e"):
		return "enum"
	default:
		return "misc"
	}
}

func ExtractStringsFromTextFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func CompareStringSets(oldStrings, newStrings []string) (added, removed []string) {
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)
	for _, s := range oldStrings {
		oldSet[s] = true
	}
	for _, s := range newStrings {
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
	return added, removed
}
