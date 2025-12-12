package depot

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	MaxCacheSize    = 20 * 1024 * 1024 * 1024
	DepotCachePath  = "/data/depot_cache"
	SteamCMDPath    = "/opt/steamcmd/steamcmd.sh"
	DownloadTimeout = 30 * time.Minute
)

type Downloader struct {
	cachePath string
	appID     int
}

func NewDownloader(appID int) *Downloader {
	cachePath := DepotCachePath
	if envPath := os.Getenv("DEPOT_CACHE_PATH"); envPath != "" {
		cachePath = envPath
	}

	os.MkdirAll(cachePath, 0755)

	return &Downloader{
		cachePath: cachePath,
		appID:     appID,
	}
}

func (d *Downloader) DownloadDepot(depotID int, manifestID string, fileFilter string) (string, error) {
	outputDir := filepath.Join(d.cachePath, fmt.Sprintf("%d_%s", depotID, manifestID))

	if _, err := os.Stat(outputDir); err == nil {
		log.Printf("Depot %d already cached at %s", depotID, outputDir)
		return outputDir, nil
	}

	loginArgs := []string{"+login", "anonymous"}
	if user := os.Getenv("STEAM_USER"); user != "" {
		if pass := os.Getenv("STEAM_PASS"); pass != "" {
			loginArgs = []string{"+login", user, pass}
		}
	}

	args := append(loginArgs,
		"+download_depot", fmt.Sprintf("%d", d.appID), fmt.Sprintf("%d", depotID), manifestID,
	)

	if fileFilter != "" {
		args = append(args, fileFilter)
	}

	args = append(args, "+quit")

	log.Printf("Downloading depot %d with manifest %s...", depotID, manifestID)

	ctx, cancel := context.WithTimeout(context.Background(), DownloadTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, SteamCMDPath, args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Download output: %s", string(output))
		return "", fmt.Errorf("failed to download depot: %w", err)
	}

	depotPath := findDepotPath(d.appID, depotID)
	if depotPath != "" {
		os.Rename(depotPath, outputDir)
		return outputDir, nil
	}

	return outputDir, nil
}

func findDepotPath(appID, depotID int) string {
	patterns := []string{
		fmt.Sprintf("/root/Steam/steamapps/content/app_%d/depot_%d", appID, depotID),
		fmt.Sprintf("/home/*/.steam/steamapps/content/app_%d/depot_%d", appID, depotID),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			return matches[0]
		}
	}
	return ""
}

func (d *Downloader) CleanupOldCache() error {
	var totalSize int64

	entries, err := os.ReadDir(d.cachePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if entry.IsDir() {
			dirSize, _ := getDirSize(filepath.Join(d.cachePath, entry.Name()))
			totalSize += dirSize
		} else {
			totalSize += info.Size()
		}
	}

	if totalSize > MaxCacheSize {
		log.Printf("Cache size %d exceeds limit %d, cleaning up...", totalSize, MaxCacheSize)
		for _, entry := range entries {
			if totalSize <= MaxCacheSize*80/100 {
				break
			}
			path := filepath.Join(d.cachePath, entry.Name())
			size, _ := getDirSize(path)
			os.RemoveAll(path)
			totalSize -= size
			log.Printf("Removed %s, freed %d bytes", entry.Name(), size)
		}
	}

	return nil
}

func getDirSize(path string) (int64, error) {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, nil
}

func (d *Downloader) GetCachedFiles(depotID int, manifestID string) ([]string, error) {
	outputDir := filepath.Join(d.cachePath, fmt.Sprintf("%d_%s", depotID, manifestID))

	var files []string
	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(outputDir, path)
			files = append(files, relPath)
		}
		return nil
	})

	return files, err
}
