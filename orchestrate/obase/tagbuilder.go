package obase

import (
	"fmt"

	"github.com/relex/slog-agent/util"
	"github.com/relex/slog-agent/util/stringtemplate"
)

// TagBuilder builds tags from key set fields
type TagBuilder struct {
	tagExpander stringtemplate.Expander
	tagBuffer   []byte
}

type tagKeyFieldIndex int

// NewTagBuilder creates a TagBuilder
func NewTagBuilder(tagTemplate string, keyNames []string) (*TagBuilder, error) {
	tagExpander, eerr := stringtemplate.NewExpander(tagTemplate, func(name string) (stringtemplate.PartProvider, error) {
		index := util.IndexOfString(keyNames, name)
		if index == -1 {
			return nil, fmt.Errorf("variable '%s' is not one of label fields", name)
		}
		return tagKeyFieldIndex(index).provideLabelSetTemplatePart, nil
	})
	if eerr != nil {
		return nil, fmt.Errorf("error creating tag expander: %w", eerr)
	}
	return &TagBuilder{
		tagExpander: tagExpander,
		tagBuffer:   make([]byte, 0, 200),
	}, nil
}

// Build constructs new tag from given label values, the length and order must match labelNames passed to NewTagBuilder
func (m *TagBuilder) Build(keyValues []string) string {
	tag, buf := m.tagExpander.RunWithBuffer(keyValues, m.tagBuffer)
	m.tagBuffer = buf
	return tag
}

func (li tagKeyFieldIndex) provideLabelSetTemplatePart(labelValues []string) string { // xx:inline
	return labelValues[li]
}
