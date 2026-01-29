package cmd

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/runner"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func New() *cobra.Command {
	c := cobra.Command{
		Use: "cron-workflow-replicator",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			configFilePath, err := cmd.Flags().GetString("config")
			if err != nil {
				return err
			}

			f, err := os.Open(configFilePath)
			if err != nil {
				return err
			}
			defer func() {
				err = f.Close()
			}()

			cfg := config.Config{}
			if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
				return err
			}

			// Extract config directory for relative path calculations
			configDir := filepath.Dir(configFilePath)

			// Validate configuration before running
			if err := cfg.ValidateConfig(configDir); err != nil {
				slog.Error("Configuration validation failed", "error", err)
				return err
			}

			r := runner.New(slog.Default())
			if err := r.Run(cmd.Context(), cfg, configDir); err != nil {
				return err
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.Flags().StringP("config", "c", "", "Path to config file")
	return &c
}
