package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tanq16/nits/internal/network"
	u "github.com/tanq16/nits/utils"
)

var uploadFlags struct {
	url string
}

var uploadCmd = &cobra.Command{
	Use:   "upload-to-http-server <file>...",
	Short: "Upload files to a nits http-server running with --upload",
	Args:  cobra.MinimumNArgs(1),
	Run:   runUpload,
}

func runUpload(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()

	server := uploadFlags.url
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}

	failed := 0
	for _, path := range args {
		name := filepath.Base(path)
		fi, err := os.Stat(path)
		if err == nil && !fi.Mode().IsRegular() {
			err = errors.New("not a regular file")
		}
		if err != nil {
			u.PrintIndentedError(fmt.Sprintf("%s: %v", name, err), err)
			failed++
			continue
		}
		m := u.NewMeter("Uploading", name, fi.Size(), u.UnitBytes)
		stored, err := network.Upload(ctx, server, path, m)
		if err != nil {
			m.Fail(err)
			failed++
			continue
		}
		m.Done()
		if stored != name {
			u.PrintIndentedSuccess(fmt.Sprintf("stored as %s", stored))
		}
	}
	if failed > 0 {
		u.PrintFatal(fmt.Sprintf("%d of %d uploads failed", failed, len(args)), nil)
	}
}

func init() {
	uploadCmd.Flags().StringVarP(&uploadFlags.url, "url", "u", "", "Server address, such as http://192.168.0.10:8080")
	uploadCmd.MarkFlagRequired("url")
}
