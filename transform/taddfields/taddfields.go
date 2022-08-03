// Package taddfields provides 'addFields' transform, which adds fields of fixed value or string template (with '$') to
// every log records, for example "message: task=$task class=$class $message" or "task_last_digit: ${task[-1:]}"
package taddfields

import (
	"fmt"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/defs"
	"github.com/relex/slog-agent/util/stringtemplate"
)

// Config for addFieldsTransform
type Config struct {
	bconfig.Header `yaml:",inline"`
	Fields         map[string]string `yaml:"fields"`
}

type addFieldsTransform struct {
	fieldPairs []addFieldPair
	buffer     []byte
}

type addFieldPair struct {
	destination    base.LogFieldLocator
	sourceTemplate stringtemplate.Expander
}

// NewTransform creates addFieldsTransform
func (c *Config) NewTransform(schema base.LogSchema, _ logger.Logger, _ base.LogCustomCounterRegistry) base.LogTransform {
	pairList := make([]addFieldPair, 0, len(c.Fields))
	for dstKey, valExpr := range c.Fields {
		dstSel := schema.MustCreateFieldLocator(dstKey)
		srcTpl, err := stringtemplate.NewExpander(valExpr, schema.CreateTemplateVariableResolver)
		if err != nil {
			panic(err)
		}
		pairList = append(pairList, addFieldPair{destination: dstSel, sourceTemplate: srcTpl})
	}
	return &addFieldsTransform{
		fieldPairs: pairList,
		buffer:     make([]byte, 0, defs.InputLogMaxMessageBytes),
	}
}

// VerifyConfig verifies addFieldsTransform config
func (c *Config) VerifyConfig(schema base.LogSchema) error {
	if len(c.Fields) == 0 {
		return fmt.Errorf(".fields is empty")
	}
	for dstKey, valExpr := range c.Fields {
		if _, err := schema.CreateFieldLocator(dstKey); err != nil {
			return fmt.Errorf(".fields[%s] is invalid: %w", dstKey, err)
		}
		if _, err := stringtemplate.NewExpander(valExpr, schema.CreateTemplateVariableResolver); err != nil {
			return fmt.Errorf(".fields[%s] has invalid template: %w", dstKey, err)
		}
	}
	return nil
}

func (tf *addFieldsTransform) Transform(record *base.LogRecord) base.FilterResult {
	buf := tf.buffer
	fields := record.Fields
	for _, pair := range tf.fieldPairs {
		var value string
		value, buf = pair.sourceTemplate.RunWithBuffer(fields, buf)
		if len(value) > 0 {
			pair.destination.Set(fields, value)
		}
	}
	tf.buffer = buf
	return base.PASS
}
