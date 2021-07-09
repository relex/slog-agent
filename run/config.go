package run

import (
	"fmt"
	"strings"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/base/bsupport"
	"github.com/relex/slog-agent/buffer"
	"github.com/relex/slog-agent/input"
	"github.com/relex/slog-agent/orchestrate"
	"github.com/relex/slog-agent/output"
	"github.com/relex/slog-agent/rewrite"
	"github.com/relex/slog-agent/transform"
	"github.com/relex/slog-agent/util"
	"gopkg.in/yaml.v3"
)

// Config defines the root of slog-agent config file
type Config struct {
	Anchors         AnchorsConfig                      `yaml:"anchors"`
	Schema          SchemaConfig                       `yaml:"schema"`
	Inputs          []bconfig.LogInputConfigHolder     `yaml:"inputs"`
	Orchestration   bconfig.OrchestratorConfigHolder   `yaml:"orchestration"`
	MetricKeys      []string                           `yaml:"metricKeys"`
	Transformations []bconfig.LogTransformConfigHolder `yaml:"transformations"`
	Buffer          bconfig.ChunkBufferConfigHolder    `yaml:"buffer"`
	Output          bconfig.LogOutputConfigHolder      `yaml:"output"`
}

// AnchorsConfig defines the anchors section in config file
//
// The section is meant to host user-defined YAML variables for other sections and doesn't need to be unmarshalled itself
type AnchorsConfig struct {
}

// SchemaConfig defines the schema section in config file
type SchemaConfig struct {
	Fields []string `yaml:"fields"`
}

func init() {
	buffer.Register()
	input.Register()
	orchestrate.Register()
	output.Register()
	rewrite.Register()
	transform.Register()
}

// ParseConfigFile loads config from the path, creates the schema and verify all configurations
func ParseConfigFile(filepath string) (Config, base.LogSchema, error) {
	cfg := Config{}
	if err := util.UnmarshalYamlFile(filepath, &cfg); err != nil {
		return cfg, base.LogSchema{}, err
	}
	if len(cfg.Schema.Fields) == 0 {
		return cfg, base.LogSchema{}, fmt.Errorf("schema: no field defined")
	}
	logger.Infof("create schema with fields: [%s]", strings.Join(cfg.Schema.Fields, ", "))
	schema, schemaErr := base.NewLogSchema(cfg.Schema.Fields)
	if schemaErr != nil {
		return cfg, schema, fmt.Errorf("schema: %w", schemaErr)
	}
	if err := bsupport.VerifyInputConfigs(cfg.Inputs, schema, "inputs"); err != nil {
		return cfg, schema, err
	}
	metricLabelNames, orchErr := cfg.Orchestration.VerifyConfig(schema)
	if orchErr != nil {
		return cfg, schema, fmt.Errorf("orchestration: %w", orchErr)
	}
	if len(cfg.MetricKeys) == 0 {
		return cfg, schema, fmt.Errorf("metricKeys is empty")
	}
	if _, err := schema.CreateFieldLocators(cfg.MetricKeys); err != nil {
		return cfg, schema, fmt.Errorf("metricKeys: %w", err)
	}
	for i, key := range cfg.MetricKeys {
		if util.IndexOfString(metricLabelNames, "key_"+key) != -1 {
			return cfg, schema, fmt.Errorf("metricKeys[%d]: key '%s' cannot exist in both .metricKeys and .orchestration", i, key)
		}
	}
	if err := bsupport.VerifyTransformConfigs(cfg.Transformations, schema, "transforms"); err != nil {
		return cfg, schema, err
	}
	if err := cfg.Buffer.VerifyConfig(); err != nil {
		return cfg, schema, fmt.Errorf("buffer: %w", err)
	}
	if err := cfg.Output.VerifyConfig(schema); err != nil {
		return cfg, schema, fmt.Errorf("output: %w", err)
	}
	return cfg, schema, nil
}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (holder AnchorsConfig) MarshalYAML() (interface{}, error) {
	return []string(nil), nil
}

// UnmarshalYAML provides custom unmarshalling for the implementations of Config
func (holder *AnchorsConfig) UnmarshalYAML(value *yaml.Node) error {
	return nil
}
