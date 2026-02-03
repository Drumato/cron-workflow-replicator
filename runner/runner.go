package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/filesystem"
	"github.com/drumato/cron-workflow-replicator/kustomize"
	"github.com/drumato/cron-workflow-replicator/structutil"
	"github.com/drumato/cron-workflow-replicator/types"
)

type Runner struct {
	logger           *slog.Logger
	fsConnector      filesystem.FileSystem
	fileReader       config.FileReader
	kustomizeManager *kustomize.Manager
}

type RunnerOption func(*Runner)

func New(logger *slog.Logger, opts ...RunnerOption) *Runner {
	fs := filesystem.NewDefaultFileSystem()
	r := Runner{
		logger:           logger,
		fsConnector:      fs,
		fileReader:       &config.DefaultFileReader{},
		kustomizeManager: kustomize.NewManager(fs),
	}
	for _, opt := range opts {
		opt(&r)
	}
	return &r
}

func WithFileSystem(fs filesystem.FileSystem) RunnerOption {
	return func(r *Runner) {
		r.fsConnector = fs
	}
}

func WithFileReader(fr config.FileReader) RunnerOption {
	return func(r *Runner) {
		r.fileReader = fr
	}
}

func WithKustomizeManager(km *kustomize.Manager) RunnerOption {
	return func(r *Runner) {
		r.kustomizeManager = km
	}
}

func (r *Runner) Run(ctx context.Context, cfg config.Config, configDir string) error {
	r.logger.Info("Runner started")

	r.logger.DebugContext(ctx, "Configuration", slog.Any("config", cfg))
	for i, unit := range cfg.Units {
		if err := r.processUnit(ctx, unit, configDir); err != nil {
			return fmt.Errorf("failed to process unit %d: %w", i, err)
		}
	}

	r.logger.Info("Runner completed successfully")
	return nil
}

func (r *Runner) processUnit(ctx context.Context, unit config.Unit, configDir string) error {
	// Calculate absolute output directory from configDir + unit.OutputDirectory
	absoluteOutputDir := filepath.Join(configDir, unit.OutputDirectory)

	// Load base CronWorkflow from manifest if provided
	baseCronWorkflow, err := unit.LoadBaseCronWorkflow(r.fileReader, configDir)
	if err != nil {
		r.logger.Error("Failed to load base CronWorkflow", "error", err)
		return fmt.Errorf("failed to load base CronWorkflow: %w", err)
	}

	if err := r.fsConnector.MkdirAll(absoluteOutputDir, 0o755); err != nil {
		r.logger.Error("Failed to create output directory", "directory", absoluteOutputDir, "error", err)
		return fmt.Errorf("failed to create output directory %s: %w", absoluteOutputDir, err)
	}

	// Track generated files for kustomize
	var generatedFiles []string
	sameFilenameCounter := map[string]int{}

	for _, value := range unit.Values {
		r.logger.DebugContext(ctx, "Processing value", slog.String("filename", value.Filename))

		var filename string
		if counter, exists := sameFilenameCounter[value.Filename]; exists {
			filename = fmt.Sprintf("%s-%d.yaml", value.Filename, counter+1)
		} else {
			filename = fmt.Sprintf("%s.yaml", value.Filename)
		}
		outputYAMLPath := filepath.Join(absoluteOutputDir, filename)
		generatedFiles = append(generatedFiles, filename)
		r.logger.DebugContext(ctx, "Generating output file", slog.String("outputYAMLPath", outputYAMLPath))

		f, err := r.fsConnector.OpenFile(outputYAMLPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			r.logger.Error("Failed to open output file", "file", outputYAMLPath, "error", err)
			return fmt.Errorf("failed to open output file %s: %w", outputYAMLPath, err)
		}

		// Start with the base CronWorkflow (deep copy to avoid modifying the original)
		cw := *baseCronWorkflow

		// Merge metadata from the value
		structutil.MergeStruct(&cw.ObjectMeta, &value.Metadata)

		// Merge spec from the value
		structutil.MergeStruct(&cw.Spec, &value.Spec)

		cleanCW := types.NewCleanCronWorkflow(&cw)
		out, err := cleanCW.ToYAMLWithIndent(unit.GetIndent())
		if err != nil {
			r.logger.Error("Failed to marshal cronworkflow to YAML", "file", outputYAMLPath, "error", err)
			return fmt.Errorf("failed to marshal cronworkflow to yaml for file %s: %w", outputYAMLPath, err)
		}

		n, err := f.Write(out)
		if err != nil {
			return fmt.Errorf("failed to write to output file %s: %w", outputYAMLPath, err)
		}
		if n < len(out) {
			return fmt.Errorf("incomplete write to output file %s: wrote %d bytes, expected %d bytes", outputYAMLPath, n, len(out))
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close output file %s: %w", outputYAMLPath, err)
		}

		// Update the counter for duplicate filenames
		sameFilenameCounter[value.Filename]++
	}

	// Update kustomization.yaml if kustomize is configured
	if unit.Kustomize != nil && unit.Kustomize.UpdateResources {
		r.logger.DebugContext(ctx, "Updating kustomization.yaml",
			slog.String("outputDir", absoluteOutputDir),
			slog.Any("generatedFiles", generatedFiles))

		if err := r.kustomizeManager.UpdateKustomization(absoluteOutputDir, generatedFiles); err != nil {
			r.logger.WarnContext(ctx, "Failed to update kustomization.yaml",
				slog.String("error", err.Error()))
			// Don't fail the entire process if kustomize update fails
		}
	}

	return nil
}
