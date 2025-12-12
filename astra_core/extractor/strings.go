package extractor

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"unicode"
)

const MinStringLength = 4

var interestingPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^CMsg[A-Z]`),
	regexp.MustCompile(`(?i)^CUser`),
	regexp.MustCompile(`(?i)^CClient`),
	regexp.MustCompile(`(?i)^CServer`),
	regexp.MustCompile(`(?i)^weapon_`),
	regexp.MustCompile(`(?i)^item_`),
	regexp.MustCompile(`(?i)^ability_`),
	regexp.MustCompile(`(?i)^hero_`),
	regexp.MustCompile(`(?i)^npc_`),
	regexp.MustCompile(`(?i)^proto`),
	regexp.MustCompile(`(?i)_proto$`),
	regexp.MustCompile(`(?i)^k_E[A-Z]`),
	regexp.MustCompile(`(?i)^DOTA_`),
	regexp.MustCompile(`(?i)^CS_`),
	regexp.MustCompile(`(?i)^game\.`),
	regexp.MustCompile(`(?i)convar`),
	regexp.MustCompile(`(?i)cvar`),
	regexp.MustCompile(`(?i)^sv_`),
	regexp.MustCompile(`(?i)^mp_`),
	regexp.MustCompile(`(?i)^cl_`),
}

type StringMatch struct {
	Value    string
	Category string
}

func ExtractStrings(filePath string) ([]string, error) {
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
	var matches []StringMatch
	seen := make(map[string]bool)

	for _, s := range strings {
		if seen[s] {
			continue
		}

		for _, pattern := range interestingPatterns {
			if pattern.MatchString(s) {
				category := categorizeString(s)
				matches = append(matches, StringMatch{
					Value:    s,
					Category: category,
				})
				seen[s] = true
				break
			}
		}
	}

	return matches
}

func categorizeString(s string) string {
	lower := strings.ToLower(s)

	switch {
	case strings.HasPrefix(lower, "cmsg") || strings.Contains(lower, "proto"):
		return "protobuf"
	case strings.HasPrefix(lower, "weapon_"):
		return "weapon"
	case strings.HasPrefix(lower, "item_"):
		return "item"
	case strings.HasPrefix(lower, "ability_") || strings.HasPrefix(lower, "hero_"):
		return "gameplay"
	case strings.HasPrefix(lower, "npc_"):
		return "npc"
	case strings.HasPrefix(lower, "sv_") || strings.HasPrefix(lower, "mp_") || strings.HasPrefix(lower, "cl_"):
		return "convar"
	case strings.HasPrefix(lower, "k_e"):
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
