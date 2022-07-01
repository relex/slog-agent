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

func init() {
	buffer.Register()
	input.Register()
	orchestrate.Register()
	output.Register()
	rewrite.Register()
	transform.Register()
}

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
// The section is meant to host user-defined YAML variables for other sections and doesn't need to be processed itself
type AnchorsConfig struct {
}

// MarshalYAML does nothing
func (holder AnchorsConfig) MarshalYAML() (interface{}, error) {
	return []string(nil), nil
}

// UnmarshalYAML does nothing
func (holder *AnchorsConfig) UnmarshalYAML(value *yaml.Node) error {
	return nil
}

// SchemaConfig defines the schema section in config file
type SchemaConfig struct {
	Fields    []string `yaml:"fields"`
	MaxFields int      `yaml:"maxFields"`
}

// ParseConfigFile loads config from the path, creates the schema and verify all configurations
func ParseConfigFile(filepath string) (Config, base.LogSchema, ConfigStats, error) {
	var conf Config
	var stats ConfigStats

	if err := util.UnmarshalYamlFile(filepath, &conf); err != nil {
		return conf, base.LogSchema{}, stats, err
	}

	var schema base.LogSchema
	if s, err := checkAndCreateSchema(conf); err == nil {
		schema = s
	} else {
		return conf, base.LogSchema{}, stats, err
	}

	statsBuilder := NewConfigStatsBuilder(&schema)
	statsBuilder.BeginTrackingFixedFields()

	if err := bsupport.VerifyInputConfigs(conf.Inputs, schema, "inputs"); err != nil {
		return conf, schema, stats, err
	}

	var orcKeys []string
	if keys, err := conf.Orchestration.Value.VerifyConfig(schema); err == nil {
		orcKeys = keys
		stats.OrchestrationKeys = keys
	} else {
		return conf, schema, stats, fmt.Errorf("orchestration: %w", err)
	}

	statsBuilder.BeginTrackingFields()

	if err := checkMetricKeys(conf, schema, orcKeys); err != nil {
		return conf, schema, stats, err
	}

	if err := bsupport.VerifyTransformConfigs(conf.Transformations, schema, "transforms"); err != nil {
		return conf, schema, stats, err
	}

	if err := conf.Buffer.Value.VerifyConfig(); err != nil {
		return conf, schema, stats, fmt.Errorf("buffer: %w", err)
	}

	if err := conf.Output.Value.VerifyConfig(schema); err != nil {
		return conf, schema, stats, fmt.Errorf("output: %w", err)
	}

	statsBuilder.Finish(&stats)
	return conf, schema, stats, nil
}

func checkAndCreateSchema(conf Config) (base.LogSchema, error) {
	if len(conf.Schema.Fields) == 0 {
		return base.LogSchema{}, fmt.Errorf("schema: no fields defined")
	}
	if conf.Schema.MaxFields == 0 {
		return base.LogSchema{}, fmt.Errorf("schema: no maxFields defined")
	}

	logger.Infof("create schema with fields: [%s]", strings.Join(conf.Schema.Fields, ", "))
	schema, schemaErr := base.NewLogSchema(conf.Schema.Fields, conf.Schema.MaxFields)
	if schemaErr != nil {
		return base.LogSchema{}, fmt.Errorf("schema: %w", schemaErr)
	}
	return schema, nil
}

func checkMetricKeys(conf Config, schema base.LogSchema, orchestrationKeys []string) error {
	if len(conf.MetricKeys) == 0 {
		return fmt.Errorf("metricKeys is empty")
	}
	if _, err := schema.CreateFieldLocators(conf.MetricKeys); err != nil {
		return fmt.Errorf("metricKeys: %w", err)
	}
	for i, key := range conf.MetricKeys {
		if util.IndexOfString(orchestrationKeys, key) != -1 {
			return fmt.Errorf("metricKeys[%d]: field '%s' cannot be listed in both .metricKeys and .orchestration/keys", i, key)
		}
	}
	return nil
}
