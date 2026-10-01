package fsSyncCmd

import (
	"fmt"

	"github.com/spf13/cobra"
	fssync "github.com/tanq16/nits/internal/interactions/fs-sync"
	u "github.com/tanq16/nits/utils"
)

type syncMode string

func (m *syncMode) String() string { return string(*m) }
func (m *syncMode) Type() string   { return "send|receive" }

func (m *syncMode) Set(v string) error {
	switch v {
	case "send", "receive":
		*m = syncMode(v)
		return nil
	}
	return fmt.Errorf("must be one of send, receive")
}

var fsSyncServeFlags struct {
	mode      syncMode
	port      int
	dir       string
	ignore    string
	enableTLS bool
	delete    bool
	dryRun    bool
}

var fsSyncClientFlags struct {
	dir      string
	ignore   string
	insecure bool
	delete   bool
	dryRun   bool
}

var FSSyncCmd = &cobra.Command{
	Use:   "fs-sync",
	Short: "One-shot bidirectional file synchronization over HTTP/HTTPS",
}

var fsSyncServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an HTTP server for file sync (use --mode to set direction)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		protocol := "http"
		if fsSyncServeFlags.enableTLS {
			protocol = "https"
		}
		u.PrintInfo(fmt.Sprintf("Starting fs-sync server (mode: %s): %s://localhost:%d directory=%s", fsSyncServeFlags.mode, protocol, fsSyncServeFlags.port, fsSyncServeFlags.dir))
		cfg := fssync.ServerConfig{
			Port:        fsSyncServeFlags.port,
			SyncDir:     fsSyncServeFlags.dir,
			IgnorePaths: fsSyncServeFlags.ignore,
			EnableTLS:   fsSyncServeFlags.enableTLS,
			Mode:        string(fsSyncServeFlags.mode),
			DeleteExtra: fsSyncServeFlags.delete,
			DryRun:      fsSyncServeFlags.dryRun,
		}
		s, err := fssync.NewServer(cfg)
		if err != nil {
			u.PrintFatal("Failed to initialize server", err)
		}
		if err := s.Run(printerCallbacks()); err != nil {
			u.PrintFatal("Failed to run server", err)
		}
	},
}

var fsSyncClientCmd = &cobra.Command{
	Use:   "client <server-url>",
	Short: "Connect to an fs-sync server and sync files (direction auto-detected)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := fssync.ClientConfig{
			ServerAddr:  args[0],
			SyncDir:     fsSyncClientFlags.dir,
			DeleteExtra: fsSyncClientFlags.delete,
			Insecure:    fsSyncClientFlags.insecure,
			DryRun:      fsSyncClientFlags.dryRun,
			IgnorePaths: fsSyncClientFlags.ignore,
		}
		c, err := fssync.NewClient(cfg)
		if err != nil {
			u.PrintFatal("Failed to initialize client", err)
		}
		if err := c.Run(printerCallbacks()); err != nil {
			u.PrintFatal("Sync failed", err)
		}
	},
}

func printerCallbacks() fssync.Callbacks {
	return fssync.Callbacks{
		OnInfo:        u.PrintInfo,
		OnGeneric:     u.PrintGeneric,
		OnItemSuccess: u.PrintIndentedSuccess,
		OnWarn:        u.PrintWarn,
		OnSuccess:     u.PrintSuccess,
		OnError:       u.PrintError,
	}
}

func init() {
	fsSyncServeFlags.mode = "send"
	fsSyncServeCmd.Flags().Var(&fsSyncServeFlags.mode, "mode", "Sync mode")
	fsSyncServeCmd.Flags().IntVarP(&fsSyncServeFlags.port, "port", "p", 8080, "Port to listen on")
	fsSyncServeCmd.Flags().StringVarP(&fsSyncServeFlags.dir, "dir", "d", ".", "Directory to sync")
	fsSyncServeCmd.Flags().StringVar(&fsSyncServeFlags.ignore, "ignore", "", "Comma-separated patterns to ignore (e.g., '.git,node_modules')")
	fsSyncServeCmd.Flags().BoolVar(&fsSyncServeFlags.enableTLS, "tls", false, "Enable HTTPS with self-signed cert")
	fsSyncServeCmd.Flags().BoolVar(&fsSyncServeFlags.delete, "delete", false, "Delete extra files not present on sender (receive mode only)")
	fsSyncServeCmd.Flags().BoolVar(&fsSyncServeFlags.dryRun, "dry-run", false, "Show what would be synced without doing it (receive mode only)")

	fsSyncClientCmd.Flags().StringVarP(&fsSyncClientFlags.dir, "dir", "d", ".", "Local directory to sync")
	fsSyncClientCmd.Flags().StringVar(&fsSyncClientFlags.ignore, "ignore", "", "Comma-separated patterns to ignore (e.g., '.git,node_modules')")
	fsSyncClientCmd.Flags().BoolVar(&fsSyncClientFlags.insecure, "insecure", false, "Skip TLS certificate verification")
	fsSyncClientCmd.Flags().BoolVar(&fsSyncClientFlags.delete, "delete", false, "Delete extra files not present on sender")
	fsSyncClientCmd.Flags().BoolVar(&fsSyncClientFlags.dryRun, "dry-run", false, "Show what would be synced without doing it")

	FSSyncCmd.AddCommand(fsSyncServeCmd)
	FSSyncCmd.AddCommand(fsSyncClientCmd)
}
