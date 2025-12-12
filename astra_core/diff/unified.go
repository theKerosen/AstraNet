package diff

import (
	"fmt"
	"strings"
)

func GenerateUnifiedDiff(oldText, newText, oldLabel, newLabel string) string {
	if oldText == "" && newText == "" {
		return ""
	}

	if oldText == "" {
		return formatAsAddition(newText)
	}

	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")

	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)

	for _, line := range oldLines {
		oldSet[line] = true
	}
	for _, line := range newLines {
		newSet[line] = true
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("--- %s\n", oldLabel))
	result.WriteString(fmt.Sprintf("+++ %s\n", newLabel))

	chunks := generateDiffChunks(oldLines, newLines)
	for _, chunk := range chunks {
		result.WriteString(chunk)
	}

	return result.String()
}

func generateDiffChunks(oldLines, newLines []string) []string {
	var chunks []string

	lcs := computeLCS(oldLines, newLines)

	oldIdx, newIdx, lcsIdx := 0, 0, 0

	var currentChunk strings.Builder
	chunkOldStart, chunkNewStart := 1, 1
	oldCount, newCount := 0, 0
	hasChanges := false

	flushChunk := func() {
		if hasChanges {
			header := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", chunkOldStart, oldCount, chunkNewStart, newCount)
			chunks = append(chunks, header+currentChunk.String())
		}
		currentChunk.Reset()
		hasChanges = false
		oldCount, newCount = 0, 0
	}

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		if lcsIdx < len(lcs) && oldIdx < len(oldLines) && newIdx < len(newLines) &&
			oldLines[oldIdx] == lcs[lcsIdx] && newLines[newIdx] == lcs[lcsIdx] {
			if hasChanges {
				currentChunk.WriteString(" " + oldLines[oldIdx] + "\n")
				oldCount++
				newCount++
			}
			oldIdx++
			newIdx++
			lcsIdx++
		} else if oldIdx < len(oldLines) && (lcsIdx >= len(lcs) || oldLines[oldIdx] != lcs[lcsIdx]) {
			if !hasChanges {
				chunkOldStart = oldIdx + 1
				chunkNewStart = newIdx + 1
			}
			hasChanges = true
			currentChunk.WriteString("-" + oldLines[oldIdx] + "\n")
			oldCount++
			oldIdx++
		} else if newIdx < len(newLines) && (lcsIdx >= len(lcs) || newLines[newIdx] != lcs[lcsIdx]) {
			if !hasChanges {
				chunkOldStart = oldIdx + 1
				chunkNewStart = newIdx + 1
			}
			hasChanges = true
			currentChunk.WriteString("+" + newLines[newIdx] + "\n")
			newCount++
			newIdx++
		}

		if oldCount+newCount > 50 {
			flushChunk()
		}
	}

	flushChunk()
	return chunks
}

func computeLCS(a, b []string) []string {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				if dp[i-1][j] > dp[i][j-1] {
					dp[i][j] = dp[i-1][j]
				} else {
					dp[i][j] = dp[i][j-1]
				}
			}
		}
	}

	lcs := make([]string, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([]string{a[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

func formatAsAddition(text string) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder
	result.WriteString("--- /dev/null\n")
	result.WriteString("+++ new\n")
	result.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
	for _, line := range lines {
		result.WriteString("+" + line + "\n")
	}
	return result.String()
}
