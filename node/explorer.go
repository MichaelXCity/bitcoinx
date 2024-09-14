package node

import (
	"context"
	"fmt"

	"github.com/blocklayerhq/bitcoinx/config"
	"github.com/blocklayerhq/bitcoinx/project"
	"github.com/blocklayerhq/bitcoinx/util"
	"github.com/pkg/errors"
)

// explorerImage defines the container image to pull for running the Bitcoinx Explorer
const explorerImage = "samalba/bitcoinx-explorer-localdev:20181204"

func startExplorer(ctx context.Context, config *config.Config, p *project.Project) error {
	cmd := []string{
		"run", "--rm",
		"-p", fmt.Sprintf("%d:8080", config.Ports.Explorer),
		"-l", "bitcoinx.cosmos.explorer",
		"-l", "bitcoinx.project=" + p.Name,
		explorerImage,
	}
	if err := util.Run(ctx, "docker", cmd...); err != nil {
		return errors.Wrap(err, "failed to start the explorer")
	}
	return nil
}
