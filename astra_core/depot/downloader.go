package depot

import (
	"context"
	"fmt"
	"io"
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
		"+@sSteamCmdForcePlatformType", "windows",
		"+download_depot", fmt.Sprintf("%d", d.appID), fmt.Sprintf("%d", depotID),
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

	// Log output to help debugging where files are stored
	log.Printf("SteamCMD Output: %s", string(output))

	if err != nil {
		return "", fmt.Errorf("failed to download depot: %w", err)
	}

	depotPath := findDepotPath(d.appID, depotID)
	if depotPath != "" {
		// Validate size before moving
		size, _ := getDirSize(depotPath)
		if size < 1000 {
			log.Printf("WARNING: Downloaded depot %d is remarkably small (%d bytes). This usually indicates authentication failure or a protected depot.", depotID, size)
		}

		if err := moveOrCopy(depotPath, outputDir); err != nil {
			return "", fmt.Errorf("failed to move depot files: %w", err)
		}
		return outputDir, nil
	}

	return outputDir, nil
}

func findDepotPath(appID, depotID int) string {
	// 1. Try known patterns first (fastest)
	patterns := []string{
		fmt.Sprintf("/root/Steam/steamapps/content/app_%d/depot_%d", appID, depotID),
		fmt.Sprintf("/home/*/.steam/steamapps/content/app_%d/depot_%d", appID, depotID),
		fmt.Sprintf("/opt/steamcmd/steamapps/content/app_%d/depot_%d", appID, depotID),
		fmt.Sprintf("/opt/steamcmd/linux32/steamapps/content/app_%d/depot_%d", appID, depotID),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		if len(matches) > 0 {
			info, err := os.Stat(matches[0])
			if err == nil && info.IsDir() {
				// Check if it's not empty
				if size, _ := getDirSize(matches[0]); size > 0 {
					return matches[0]
				}
			}
		}
	}

	// 2. Fallback: Recursive search in common roots
	roots := []string{"/opt/steamcmd", "/root", "/data"}
	targetName := fmt.Sprintf("depot_%d", depotID)

	for _, root := range roots {
		found := ""
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && info.Name() == targetName {
				// Verify parent structure to avoid false positives if possible, or just accept it
				found = path
				return fmt.Errorf("found") // Stop walking
			}
			return nil
		})

		if found != "" {
			log.Printf("Found depot %d via recursive search at: %s", depotID, found)
			return found
		}
	}

	// 3. Debug: Log structure of /opt/steamcmd to help diagnosis
	log.Println("DEBUG: Dumping /opt/steamcmd structure:")
	filepath.Walk("/opt/steamcmd", func(path string, info os.FileInfo, err error) error {
		if err == nil {
			log.Println(path)
		}
		return nil
	})

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

func moveOrCopy(src, dst string) error {
	// Try atomic rename first
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename fails (likely cross-device), fallback to copy+delete
	log.Printf("Rename failed (%v), falling back to copy...", err)

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()

		out, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer out.Close()

		if _, err := io.Copy(out, in); err != nil {
			return err
		}

		return out.Chmod(info.Mode())
	})

	if err != nil {
		return err
	}

	// Remove source after successful copy
	return os.RemoveAll(src)
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
