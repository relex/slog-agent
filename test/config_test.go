package test

import (
	"os"
	"testing"

	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/testdata"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestConfigDump(t *testing.T) {
	if !util.IsTestGenerationMode() {
		return
	}
	t.Log("regenerate config dump...")
	config, _, cstats, cerr := run.ParseConfigFile(testdata.GetConfigPath())
	assert.Nil(t, cerr)
	cstats.Log(logger.WithField("test", t.Name()))

	configDump, e := util.MarshalYaml(config)
	assert.Nil(t, e)
	assert.Nil(t, os.WriteFile(testdata.GetConfigDumpPath(), []byte(configDump), 0644))
}

func TestConfigParsing(t *testing.T) {
	if util.IsTestGenerationMode() {
		return
	}
	expectedDump, rerr := os.ReadFile(testdata.GetConfigDumpPath())
	assert.Nil(t, rerr)

	config, _, cstats, cerr := run.ParseConfigFile(testdata.GetConfigPath())
	assert.Nil(t, cerr)
	cstats.Log(logger.WithField("test", t.Name()))

	configDump, e := util.MarshalYaml(config)
	assert.Nil(t, e)

	assert.Equal(t, string(expectedDump), configDump)
}
