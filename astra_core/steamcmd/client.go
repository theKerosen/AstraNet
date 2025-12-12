package steamcmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	initialized bool
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) Start() error {
	c.initialized = true
	return nil
}

func (c *Client) runCommand(args ...string) (string, error) {
	loginArgs := []string{"+login", "anonymous"}

	if user := os.Getenv("STEAM_USER"); user != "" {
		if pass := os.Getenv("STEAM_PASS"); pass != "" {
			loginArgs = []string{"+login", user, pass}
			// Se tiver Steam Guard, pode precisar de +login user pass code
			// Mas para automatização geralmente se usa contas sem 2FA ou com Sentry file salvo
		}
	}

	fullArgs := append(loginArgs, args...)
	fullArgs = append(fullArgs, "+quit")

	// Avoid logging password
	logArgs := make([]string, len(fullArgs))
	copy(logArgs, fullArgs)
	if len(loginArgs) == 3 {
		logArgs[2] = "*****"
	}
	log.Printf("Executing steamcmd with args: %v", logArgs)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // Increased timeout for authenticated downloads
	defer cancel()

	cmd := exec.CommandContext(ctx, "/opt/steamcmd/steamcmd.sh", fullArgs...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() != nil {
		return string(output), fmt.Errorf("command timed out")
	}

	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func (c *Client) LoginAnonymous() error {
	return nil
}

func (c *Client) AppInfoUpdate(appID int) error {
	_, err := c.runCommand("+app_info_update", "1")
	return err
}

func (c *Client) AppInfoPrint(appID int) (string, error) {
	output, err := c.runCommand(
		"+app_info_update", "1",
		"+app_info_print", fmt.Sprintf("%d", appID),
	)
	if err != nil {
		if strings.Contains(output, fmt.Sprintf("\"%d\"", appID)) {
			return output, nil
		}
		return output, err
	}
	return output, nil
}

func (c *Client) DownloadDepot(appID, depotID int, manifestID string, fileFilter string) (string, error) {
	args := []string{"+download_depot", fmt.Sprintf("%d", appID), fmt.Sprintf("%d", depotID), manifestID}
	if fileFilter != "" {
		args = append(args, fileFilter)
	}
	return c.runCommand(args...)
}

func (c *Client) Quit() error {
	return nil
}
