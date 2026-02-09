package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/runner"
	"github.com/drumato/cron-workflow-replicator/template"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func New() *cobra.Command {
	c := cobra.Command{
		Use: "cron-workflow-replicator",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMain(cmd, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	c.Flags().StringP("config", "c", "", "Path to config file")
	c.Flags().String("values", "", "Path to values file for template rendering")

	// Add render-config subcommand
	renderConfigCmd := &cobra.Command{
		Use:   "render-config",
		Short: "Render config template and output to stdout",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRenderConfig(cmd, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	renderConfigCmd.Flags().StringP("config", "c", "", "Path to config file")
	renderConfigCmd.Flags().String("values", "", "Path to values file for template rendering")
	c.AddCommand(renderConfigCmd)

	return &c
}

func runMain(cmd *cobra.Command, args []string) (err error) {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	valuesFilePath, err := cmd.Flags().GetString("values")
	if err != nil {
		return err
	}

	// Load and potentially render the config
	configContent, err := loadConfigWithTemplate(configFilePath, valuesFilePath)
	if err != nil {
		return err
	}

	cfg := config.Config{}
	if err := yaml.Unmarshal(configContent, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
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
}

func runRenderConfig(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	valuesFilePath, err := cmd.Flags().GetString("values")
	if err != nil {
		return err
	}

	// Load and render the config
	configContent, err := loadConfigWithTemplate(configFilePath, valuesFilePath)
	if err != nil {
		return err
	}

	// Output the rendered config to stdout
	fmt.Print(string(configContent))
	return nil
}

func loadConfigWithTemplate(configFilePath, valuesFilePath string) ([]byte, error) {
	if valuesFilePath == "" {
		// No template rendering, load config directly
		return os.ReadFile(configFilePath)
	}

	// Render the config as a template
	renderer := template.New(slog.Default())
	return renderer.RenderConfig(configFilePath, valuesFilePath)
}
