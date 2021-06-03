package tblock

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/transform/taddfields"
	"github.com/relex/slog-agent/transform/tdelfields"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestBlockTransform(t *testing.T) {
	bconfig.RegisterLogTransformConfigConstructors(map[string]func() bconfig.LogTransformConfig{
		"addFields": func() bconfig.LogTransformConfig { return &taddfields.Config{} },
		"delFields": func() bconfig.LogTransformConfig { return &tdelfields.Config{} },
	})
	schema := base.MustNewLogSchema([]string{"animal", "name"})
	iAnimal := 0
	iName := 1
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: block
steps:
  - type: addFields
    fields:
      animal: $name
  - type: delFields
    keys: [name]
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		record := schema.NewTestRecord1(base.LogFields{"", "Dog"})
		status := tf.Transform(record)
		assert.Equal(t, base.PASS, status)
		assert.Equal(t, "Dog", record.Fields[iAnimal])
		assert.Equal(t, "", record.Fields[iName])
	}
}
