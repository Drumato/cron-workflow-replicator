package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/drumato/cron-workflow-replicator/config"
	"github.com/drumato/cron-workflow-replicator/filesystem"
	"github.com/drumato/cron-workflow-replicator/structutil"
	kyaml "sigs.k8s.io/yaml"
)

type Runner struct {
	logger      *slog.Logger
	fsConnector filesystem.FileSystem
	fileReader  config.FileReader
}

type RunnerOption func(*Runner)

func New(logger *slog.Logger, opts ...RunnerOption) *Runner {
	r := Runner{
		logger:      logger,
		fsConnector: filesystem.NewDefaultFileSystem(),
		fileReader:  &config.DefaultFileReader{},
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

func (r *Runner) Run(ctx context.Context, cfg config.Config) error {
	r.logger.Info("Runner started")

	r.logger.DebugContext(ctx, "Configuration", slog.Any("config", cfg))
	for _, unit := range cfg.Units {
		if err := r.processUnit(ctx, unit); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) processUnit(ctx context.Context, unit config.Unit) error {
	// Load base CronWorkflow from manifest if provided
	baseCronWorkflow, err := unit.LoadBaseCronWorkflow(r.fileReader)
	if err != nil {
		return fmt.Errorf("failed to load base CronWorkflow: %w", err)
	}

	sameFilenameCounter := map[string]int{}
	for _, value := range unit.Values {
		r.logger.DebugContext(ctx, "Processing value", slog.String("filename", value.Filename))

		if err := r.fsConnector.MkdirAll(unit.OutputDirectory, 0o755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", unit.OutputDirectory, err)
		}

		var filename string
		if counter, exists := sameFilenameCounter[value.Filename]; exists {
			filename = fmt.Sprintf("%s-%d.yaml", value.Filename, counter+1)
		} else {
			filename = fmt.Sprintf("%s.yaml", value.Filename)
		}
		outputYAMLPath := filepath.Join(unit.OutputDirectory, filename)
		r.logger.DebugContext(ctx, "Generating output file", slog.String("outputYAMLPath", outputYAMLPath))

		f, err := r.fsConnector.OpenFile(outputYAMLPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("failed to open output file %s: %w", outputYAMLPath, err)
		}

		// Start with the base CronWorkflow (deep copy to avoid modifying the original)
		cw := *baseCronWorkflow

		// Merge metadata from the value
		structutil.MergeStruct(&cw.ObjectMeta, &value.Metadata)

		// Merge spec from the value
		structutil.MergeStruct(&cw.Spec, &value.Spec)

		out, err := kyaml.Marshal(cw)
		if err != nil {
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
	}

	return nil
}
