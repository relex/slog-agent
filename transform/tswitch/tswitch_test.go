package tswitch

import (
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/base/bconfig"
	"github.com/relex/slog-agent/transform/taddfields"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestSwitchConfig(t *testing.T) {
	bconfig.RegisterLogTransformConfigConstructors(map[string]func() bconfig.LogTransformConfig{
		"addFields": func() bconfig.LogTransformConfig { return &taddfields.Config{} },
	})
	schema := base.MustNewLogSchema([]string{"type", "cost"})
	{
		c := &Config{}
		assert.Nil(t, util.UnmarshalYamlString(`
type: switch
cases:
  - match:
      type: Fruit
    then:
      - type: addFields
        fields:
          cost: 10
  - match:
      type: Animal
    then:
      - type: addFields
        fields:
          cost: 20
`, c))
		assert.Nil(t, c.VerifyConfig(schema))
		tf := c.NewTransform(schema, logger.Root(), nil)
		{
			record := schema.NewTestRecord1(base.LogFields{"Fruit", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "10", record.Fields[1])
		}
		{
			record := schema.NewTestRecord1(base.LogFields{"Animal", ""})
			status := tf.Transform(record)
			assert.Equal(t, base.PASS, status)
			assert.Equal(t, "20", record.Fields[1])
		}
	}
}
