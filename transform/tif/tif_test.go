package tif

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

func TestIfTransform(t *testing.T) {
	bconfig.RegisterConfigConstructors(bconfig.LogTransformConfigCreatorTable{
		"addFields": func() bconfig.LogTransformConfig { return &taddfields.Config{} },
		"delFields": func() bconfig.LogTransformConfig { return &tdelfields.Config{} },
	})
	schema := base.MustNewLogSchema([]string{"type", "name", "fruit"})
	iType := 0
	iName := 1
	iFruit := 2
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: if
match:
  type: Fruit
then:
  - type: addFields
    fields:
      fruit: $name
  - type: delFields
    keys: [type, name]
`, c))
		tf := c.NewTransform(schema, logger.Root(), nil)
		{
			record := schema.NewTestRecord1(base.LogFields{"Fruit", "Apple", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "", record.Fields[iType])
			assert.Equal(t, "", record.Fields[iName])
			assert.Equal(t, "Apple", record.Fields[iFruit])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"Animal", "Cat", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "Animal", record.Fields[iType])
			assert.Equal(t, "Cat", record.Fields[iName])
			assert.Equal(t, "", record.Fields[iFruit])
		}
	}
}
