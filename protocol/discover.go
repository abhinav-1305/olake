package protocol

import (
	"errors"
	"fmt"

	"github.com/datazip-inc/olake/types"
	"github.com/datazip-inc/olake/utils"
	"github.com/spf13/cobra"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "discover command",
	PreRunE: func(_ *cobra.Command, _ []string) error {
		if configPath == "" {
			return fmt.Errorf("--config not passed")
		}

		if err := utils.UnmarshalFile(configPath, connector.GetConfigRef()); err != nil {
			return err
		}

		if streamsPath != "" {
			if err := utils.UnmarshalFile(streamsPath, &catalog); err != nil {
				return fmt.Errorf("failed to read streams from %s: %w", streamsPath, err)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		err := connector.Setup(cmd.Context())
		if err != nil {
			return err
		}
		streams, err := connector.Discover(cmd.Context())
		if err != nil {
			return err
		}

		if len(streams) == 0 {
			return errors.New("no streams found in connector")
		}

		types.LogCatalog(streams, catalog)
		return nil
	},
}
