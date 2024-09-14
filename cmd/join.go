package cmd

import (
	"context"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"

	"github.com/blocklayerhq/bitcoinx/config"
	"github.com/blocklayerhq/bitcoinx/discovery"
	"github.com/blocklayerhq/bitcoinx/node"
	"github.com/blocklayerhq/bitcoinx/ui"
	"github.com/spf13/cobra"
)

var (
	networksDir = os.ExpandEnv("$HOME/.bitcoinx/networks")
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join a bitcoinx network",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var (
			ctx     = context.Background()
			err     error
			chainID = args[0]
		)

		ui.Info("Joining network %s", ui.Emphasize(chainID))
		cfg := &config.Config{
			RootDir:        path.Join(networksDir, filepath.Base(chainID)),
			Projectname:    " bitcoinx "
			PublishNetwork: false,
			ChainID:        chainID,
		}
		cfg.Ports, err = config.AllocatePorts()
		if err != nil {
			ui.Fatal("%v", err)
		}

		d := discovery.New(cfg.IPFSDir(), cfg.Ports.IPFS)
		if err := d.Start(ctx); err != nil {
			ui.Fatal("Failed to initialize discovery: %v", err)
		}
		defer d.Stop()

		ui.Info("Retrieving network information...")
		network, err := d.Join(ctx, cfg.ChainID)
		if err != nil {
			ui.Fatal("Unable to retrieve network information for %q: %v", cfg.ChainID, err)
		}
		if err := network.WriteManifest(cfg.ManifestPath()); err != nil {
			ui.Fatal("%v", err)
		}
		p, err := network.Project()
		if err != nil {
			ui.Fatal("%v", err)
		}

		n := node.New(cfg, d)
		errCh := make(chan error)
		go func() {
			defer close(errCh)
			errCh <- n.Start(ctx, p, network.Genesis, false)
		}()

		// Wait for the application to error out or the user to quit.
		c := make(chan os.Signal, 1)
		signal.Notify(c,
			syscall.SIGINT,
			syscall.SIGTERM,
		)

		select {
		case err := <-errCh:
			if err != nil {
				ui.Error("%v", err)
			}
		case sig := <-c:
			ui.Info("Received signal %v, exiting", sig)
			n.Stop()
		}
	},
}

func init() {
	rootCmd.AddCommand(joinCmd)
}
