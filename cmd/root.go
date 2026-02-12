package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"

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

	// Add list subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List files in output directories and show their management status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, args)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	listCmd.Flags().StringP("config", "c", "", "Path to config file")
	listCmd.Flags().String("values", "", "Path to values file for template rendering")
	c.AddCommand(listCmd)

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

func runList(cmd *cobra.Command, args []string) error {
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

	// Validate configuration before listing
	if err := cfg.ValidateConfig(configDir); err != nil {
		slog.Error("Configuration validation failed", "error", err)
		return err
	}

	// Process each unit
	for _, unit := range cfg.Units {
		if err := listFilesInUnit(unit, configDir); err != nil {
			return fmt.Errorf("failed to list files for unit: %w", err)
		}
	}

	return nil
}

func listFilesInUnit(unit config.Unit, configDir string) error {
	// Calculate absolute output directory path
	outputDir := unit.OutputDirectory
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(configDir, outputDir)
	}

	// Generate expected managed files list
	expectedFiles := generateExpectedFiles(unit)

	// Get actual files in the directory
	actualFiles, err := getActualFiles(outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory %s: %w", outputDir, err)
	}

	// Print output directory header
	fmt.Printf("OutputDirectory: %s\n", unit.OutputDirectory)

	// Create a map for quick lookup of expected files
	expectedMap := make(map[string]bool)
	for _, file := range expectedFiles {
		expectedMap[file] = true
	}

	// Separate files into managed and unmanaged
	var managedFiles []string
	var unmanagedFiles []string

	for _, file := range actualFiles {
		if expectedMap[file] {
			managedFiles = append(managedFiles, file)
		} else {
			unmanagedFiles = append(unmanagedFiles, file)
		}
	}

	// Print managed files
	if len(managedFiles) > 0 {
		fmt.Println("Managed:")
		for _, file := range managedFiles {
			fmt.Printf("- %s\n", file)
		}
	}

	// Print unmanaged files
	if len(unmanagedFiles) > 0 {
		fmt.Println("Unmanaged:")
		for _, file := range unmanagedFiles {
			fmt.Printf("- %s\n", file)
		}
	}

	fmt.Println()
	return nil
}

func generateExpectedFiles(unit config.Unit) []string {
	var files []string
	filenameCounter := map[string]int{}

	// Generate YAML files from values
	for _, value := range unit.Values {
		var filename string
		if counter, exists := filenameCounter[value.Filename]; exists {
			filename = fmt.Sprintf("%s-%d.yaml", value.Filename, counter+1)
		} else {
			filename = fmt.Sprintf("%s.yaml", value.Filename)
		}
		files = append(files, filename)
		filenameCounter[value.Filename]++
	}

	// Add kustomization.yaml if kustomize is configured
	if unit.Kustomize != nil && unit.Kustomize.UpdateResources {
		files = append(files, "kustomization.yaml")
	}

	return files
}

func getActualFiles(outputDir string) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	// Sort files for consistent output
	sort.Strings(files)
	return files, nil
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
