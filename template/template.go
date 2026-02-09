package template

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"text/template"

	"gopkg.in/yaml.v3"
)

// TemplateRenderer provides template rendering functionality for config files
type TemplateRenderer struct {
	logger *slog.Logger
}

// TemplateData contains all variables available in templates
type TemplateData struct {
	Var map[string]interface{} // Variables loaded from the values file
}

// New creates a new TemplateRenderer instance
func New(logger *slog.Logger) *TemplateRenderer {
	return &TemplateRenderer{
		logger: logger,
	}
}

// RenderConfig renders a config file as a template using the provided values file
func (tr *TemplateRenderer) RenderConfig(configPath, valuesPath string) ([]byte, error) {
	// Read the config file (template)
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Load variables from the values file
	variables, err := tr.loadVariables(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load variables from %s: %w", valuesPath, err)
	}

	// Create template data
	templateData := &TemplateData{
		Var: variables,
	}

	// Render the template
	renderedContent, err := tr.renderTemplate(string(configContent), templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return []byte(renderedContent), nil
}

// RenderConfigFromReader renders a config template from an io.Reader using the provided values file
func (tr *TemplateRenderer) RenderConfigFromReader(configReader io.Reader, valuesPath string) ([]byte, error) {
	// Read the config content from the reader
	configContent, err := io.ReadAll(configReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from reader: %w", err)
	}

	// Load variables from the values file
	variables, err := tr.loadVariables(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load variables from %s: %w", valuesPath, err)
	}

	// Create template data
	templateData := &TemplateData{
		Var: variables,
	}

	// Render the template
	renderedContent, err := tr.renderTemplate(string(configContent), templateData)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	return []byte(renderedContent), nil
}

// loadVariables loads variables from a YAML file into a map[string]interface{}
func (tr *TemplateRenderer) loadVariables(valuesPath string) (map[string]interface{}, error) {
	valuesContent, err := os.ReadFile(valuesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file: %w", err)
	}

	var variables map[string]interface{}
	if err := yaml.Unmarshal(valuesContent, &variables); err != nil {
		return nil, fmt.Errorf("failed to parse values file as YAML: %w", err)
	}

	tr.logger.Debug("loaded variables from values file",
		"valuesPath", valuesPath,
		"variableCount", len(variables))

	return variables, nil
}

// renderTemplate renders a template string with the provided data
func (tr *TemplateRenderer) renderTemplate(templateStr string, data *TemplateData) (string, error) {
	tmpl, err := template.New("config").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// HasTemplateVars checks if the config content contains template variables
// This is a simple heuristic check for template syntax
func HasTemplateVars(configContent string) bool {
	return bytes.Contains([]byte(configContent), []byte("{{")) && bytes.Contains([]byte(configContent), []byte("}}"))
}