package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/download"
	"github.com/tanq16/nits/internal/ghrelease"
	u "github.com/tanq16/nits/utils"
)

var ghReleaseFlags struct {
	output    string
	proxy     string
	userAgent string
	headers   []string
	token     string
	asset     string
	manual    bool
}

var gitHubReleaseCmd = &cobra.Command{
	Use:     "github-release <owner/repo or url>",
	Aliases: []string{"ghr", "ghrelease"},
	Short:   "Download a GitHub release asset",
	Args:    cobra.ExactArgs(1),
	Run:     runGitHubRelease,
}

func runGitHubRelease(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	owner, repo, err := ghrelease.ParseRepo(args[0])
	if err != nil {
		u.PrintFatal("invalid GitHub repository", err)
	}

	headers := download.ParseHeaders(ghReleaseFlags.headers)
	if token := cmp.Or(ghReleaseFlags.token, os.Getenv("GITHUB_TOKEN")); token != "" {
		if headers == nil {
			headers = map[string]string{}
		}
		headers["Authorization"] = "Bearer " + token
	}
	client, err := download.NewClient(download.ClientConfig{
		ProxyURL:  ghReleaseFlags.proxy,
		UserAgent: ghReleaseFlags.userAgent,
		Headers:   headers,
	})
	if err != nil {
		u.PrintFatal("failed to create HTTP client", err)
	}

	u.PrintRunning("checking latest release")
	release, err := ghrelease.Latest(ctx, owner, repo, client)
	u.ClearLines(1)
	if err != nil {
		u.PrintFatal("failed to fetch release info", err)
	}

	asset, ok := pickAsset(release)
	if !ok {
		return
	}

	output := ghReleaseFlags.output
	if output == "" {
		output = asset.Name
	}

	cfg := download.Config{
		URL:         asset.URL,
		OutputPath:  output,
		Connections: 1,
		ProxyURL:    ghReleaseFlags.proxy,
		UserAgent:   ghReleaseFlags.userAgent,
		Headers:     headers,
	}
	plan, dlClient, err := download.Prepare(ctx, cfg)
	if download.AlreadyComplete(err) {
		u.PrintSuccess(download.AlreadyCompletePath(err) + " already exists")
		return
	}
	if err != nil {
		u.PrintFatal("failed to start download", err)
	}

	m := u.NewMeter("Downloading", filepath.Base(plan.OutputPath), plan.Size, u.UnitBytes)
	if err := plan.Execute(ctx, dlClient, m); err != nil {
		m.Fail(err)
		u.PrintFatal("download failed", err)
	}
	m.Done()
}

func pickAsset(release ghrelease.Release) (ghrelease.Asset, bool) {
	if len(release.Assets) == 0 {
		u.PrintFatal(fmt.Sprintf("release %s has no assets", release.Tag), nil)
	}
	if ghReleaseFlags.asset != "" {
		asset, ok := ghrelease.FindAsset(release.Assets, ghReleaseFlags.asset)
		if !ok {
			u.PrintFatal(fmt.Sprintf("no asset matching %q in release %s", ghReleaseFlags.asset, release.Tag), nil)
		}
		return asset, true
	}
	if !ghReleaseFlags.manual {
		asset, ok := ghrelease.SelectForPlatform(release.Assets)
		if !ok {
			u.PrintFatal(fmt.Sprintf("no asset found for %s/%s, pass --asset or --manual", runtime.GOOS, runtime.GOARCH), nil)
		}
		return asset, true
	}

	options := make([]string, len(release.Assets))
	for i, asset := range release.Assets {
		options[i] = fmt.Sprintf("%s (%s)", asset.Name, formatAssetSize(asset.Size))
	}
	idx, err := u.PromptSelect(fmt.Sprintf("Release %s", release.Tag), options)
	if errors.Is(err, u.ErrNoTerminal) {
		u.PrintFatal("github-release --manual needs a terminal, or pass --asset", nil)
	}
	if err != nil {
		u.PrintFatal("TUI error", err)
	}
	if idx < 0 {
		return ghrelease.Asset{}, false
	}
	return release.Assets[idx], true
}

func formatAssetSize(n int64) string {
	const k = 1024.0
	v := float64(n)
	switch {
	case v < k:
		return fmt.Sprintf("%d B", n)
	case v < k*k:
		return fmt.Sprintf("%.1f KB", v/k)
	case v < k*k*k:
		return fmt.Sprintf("%.1f MB", v/(k*k))
	default:
		return fmt.Sprintf("%.1f GB", v/(k*k*k))
	}
}

func init() {
	gitHubReleaseCmd.Flags().StringVarP(&ghReleaseFlags.output, "output", "o", "", "Output file path")
	gitHubReleaseCmd.Flags().StringVarP(&ghReleaseFlags.proxy, "proxy", "p", "", "HTTP/HTTPS proxy URL")
	gitHubReleaseCmd.Flags().StringVarP(&ghReleaseFlags.userAgent, "user-agent", "a", "nits", "User agent")
	gitHubReleaseCmd.Flags().StringArrayVarP(&ghReleaseFlags.headers, "header", "H", nil, "Custom header as Key: Value (repeatable)")
	gitHubReleaseCmd.Flags().StringVar(&ghReleaseFlags.token, "token", "", "GitHub token (or GITHUB_TOKEN env)")
	gitHubReleaseCmd.Flags().StringVar(&ghReleaseFlags.asset, "asset", "", "Release asset name to download")
	gitHubReleaseCmd.Flags().BoolVar(&ghReleaseFlags.manual, "manual", false, "Select the release asset interactively")
	gitHubReleaseCmd.MarkFlagsMutuallyExclusive("manual", "asset")
}
