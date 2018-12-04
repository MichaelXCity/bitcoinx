package cmd

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/blocklayerhq/chainkit/discovery"
	"github.com/blocklayerhq/chainkit/project"
	"github.com/blocklayerhq/chainkit/ui"
	"github.com/spf13/cobra"
)

// ExplorerImage defines the container image to pull for running the Cosmos Explorer
const ExplorerImage = "samalba/cosmos-explorer-localdev:20181204"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the application",
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		p, err := project.Load(getCwd(cmd))
		if err != nil {
			ui.Fatal("%v", err)
		}
		join, err := cmd.Flags().GetString("join")
		if err != nil {
			ui.Fatal("%v", err)
		}
		start(p, join)
	},
}

func init() {
	startCmd.Flags().String("cwd", ".", "specifies the current working directory")
	startCmd.Flags().String("join", "", "join a network")

	rootCmd.AddCommand(startCmd)
}

func startExplorer(ctx context.Context, p *project.Project) {
	cmd := []string{
		"run", "--rm",
		"-p", fmt.Sprintf("%d:8080", p.Ports.Explorer),
		ExplorerImage,
	}
	if err := docker(ctx, p, cmd...); err != nil {
		ui.Fatal("Failed to start the Explorer: %v", err)
	}
}

func start(p *project.Project, join string) {
	ctx, cancel := context.WithCancel(context.Background())
	ui.Info("Starting %s", p.Name)

	// Initialize if needed.
	if err := initialize(ctx, p); err != nil {
		ui.Fatal("Initialization failed: %v", err)
	}

	s := discovery.New(p.IPFSDir(), p.Ports.IPFS)
	if err := s.Start(ctx); err != nil {
		ui.Fatal("%v", err)
	}
	defer s.Stop()

	for _, addr := range s.ListenAddresses() {
		ui.Verbose("IPFS Swarm listening on %s", addr)
	}

	for _, addr := range s.AnnounceAddresses() {
		ui.Verbose("IPFS Swarm announcing %s", addr)
	}

	// Start a network.
	if join == "" {
		chainID, err := s.Announce(ctx, p.GenesisPath())
		if err != nil {
			ui.Fatal("%v", err)
		}
		ui.Success("Network is live at: %v", chainID)
	} else {
		ui.Info("Joining network %s", join)
		genesisData, peerCh, err := s.Join(ctx, join)
		if err != nil {
			ui.Fatal("%v", err)
		}

		ui.Success("Retrieved genesis data")

		if err := ioutil.WriteFile(p.GenesisPath(), genesisData, 0644); err != nil {
			ui.Fatal("Unable to write genesis file: %v", err)
		}

		peer := <-peerCh
		ui.Info("Peer: %v", peer.Addrs)
	}

	ui.Success("Application is live at:     %s", ui.Emphasize(fmt.Sprintf("http://localhost:%d/", p.Ports.TendermintRPC)))
	ui.Success("Cosmos Explorer is live at: %s", ui.Emphasize(fmt.Sprintf("http://localhost:%d/?rpc_port=%d", p.Ports.Explorer, p.Ports.TendermintRPC)))
	defer cancel()
	go startExplorer(ctx, p)
	if err := dockerRun(ctx, p, "start"); err != nil {
		ui.Fatal("Failed to start the application: %v", err)
	}
}
