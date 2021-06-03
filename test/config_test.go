package test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/relex/slog-agent/run"
	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfigDump(t *testing.T) {
	if util.IndexOfString(os.Args, TestOutputGenerationArg) == -1 {
		return
	}
	t.Log("regenerate config dump...")
	config, _, cerr := run.LoadConfigFile("../testdata/config_sample.yml")
	assert.Nil(t, cerr)
	configDump, e := testMarshal(config)
	assert.Nil(t, e)
	assert.Nil(t, ioutil.WriteFile("../testdata/config_sample_dump.yml", configDump, 0644))
}

func TestConfigParsing(t *testing.T) {
	expectedDump, eerr := ioutil.ReadFile("../testdata/config_sample_dump.yml")
	assert.Nil(t, eerr)

	config, _, err := run.LoadConfigFile("../testdata/config_sample.yml")
	assert.Nil(t, err)
	configDump, e := testMarshal(config)
	assert.Nil(t, e)

	assert.Equal(t, string(expectedDump), string(configDump))
}

func testMarshal(config *run.Config) ([]byte, error) {
	writer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	if err := encoder.Encode(config); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}
