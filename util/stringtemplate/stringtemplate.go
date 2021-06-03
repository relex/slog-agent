// Package stringtemplate provides string expansion by pre-compiled templates, for example:
//
//     vehicle1 := map[string]string{"type": "Car", "color": "Red", "model": "X001"}
//     tag := NewStringMapExpander("${color[:1]}-$type").Expand(vehicle1)
//     // tag == "R-Car"
//
// Only named fields are supported
package stringtemplate

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/relex/slog-agent/util"
)

// RecordType is the type of records as source of string expansion
type RecordType = []string

// Expander provides a precompiled and generic alternative to regexp.Regexp.Expand()
// Expander instances contain no buffer and may be copied or concurrently used.
type Expander struct {
	partProviders []PartProvider
}

// VariableResolverCreator represents a function to create PartProvider from variable name
type VariableResolverCreator func(name string) (PartProvider, error)

// PartProvider represents a function to provide the value for a given part, from the source of template
type PartProvider func(source RecordType) string

// Empty is an empty template
var Empty = Expander{nil}

var partRegex = regexp.MustCompile(`(\$\w+|\$\{\w+[^}]*\}|[^$]+)`)

// variableExpressionRegex supports simple substring inside "${VARIABLE}", for example "name[-5:]"
var variableExpressionRegex = regexp.MustCompile(`^(?P<name>\w+)(\[(?P<start>-?[0-9]+)?:(?P<end>-?[0-9]+)?\])?$`)

var capturedNameIndex int
var capturedStartIndex int
var capturedEndIndex int

func init() {
	capturedNameIndex = variableExpressionRegex.SubexpIndex("name")
	capturedStartIndex = variableExpressionRegex.SubexpIndex("start")
	capturedEndIndex = variableExpressionRegex.SubexpIndex("end")
}

// NewExpander creates an Expander that acts like regexp.Regexp.Expand()
func NewExpander(template string, createVariableResolver VariableResolverCreator) (Expander, error) {
	if si := strings.Index(template, "$$"); si != -1 {
		return Empty, fmt.Errorf("escaping of $ at index %d of '%s' is unsupported", si, template)
	}
	parts := partRegex.FindAllString(template, -1)
	partProviders := make([]PartProvider, len(parts))
	extractedLen := 0
	for i, p := range parts {
		extractedLen += len(p)
		if p[0] != '$' {
			partProviders[i] = newPartResolverForString(p)
			continue
		}
		if p[1] == '{' && p[len(p)-1] == '}' {
			vexpr := p[2 : len(p)-1]
			vexprSubmatches := variableExpressionRegex.FindStringSubmatch(vexpr)
			if vexprSubmatches == nil {
				return Empty, fmt.Errorf("unrecognized variable expression '${%s}'", vexpr)
			}
			vname := vexprSubmatches[capturedNameIndex]
			vprovider, err := createVariableResolver(vname)
			if err != nil {
				return Empty, fmt.Errorf("error creating resolver for $%s: %w", vname, err)
			}
			partProviders[i] = createVariableExpressionSolver(vprovider, vexprSubmatches)
		} else {
			vname := p[1:]
			vprovider, err := createVariableResolver(vname)
			if err != nil {
				return Empty, fmt.Errorf("error creating resolver for $%s: %w", vname, err)
			}
			partProviders[i] = vprovider
		}
	}
	if extractedLen != len(template) {
		return Empty, fmt.Errorf("unenclosed variable quotes: '%s'", template)
	}
	return Expander{
		partProviders: partProviders,
	}, nil
}

// Run expands the template with given fields
func (tmpl Expander) Run(fields RecordType) string {
	// shortcut for most scenarios
	if len(tmpl.partProviders) == 1 {
		return tmpl.partProviders[0](fields)
	}
	buf := make([]byte, 0, 100)
	for _, provider := range tmpl.partProviders {
		part := provider(fields)
		buf = append(buf, part...)
	}
	return util.DeepCopyStringFromBytes(buf)
}

// RunWithBuffer expands the template with given fields and existing buffer
// Returns string result and the given buffer reset to zero length
func (tmpl Expander) RunWithBuffer(fields RecordType, buffer []byte) (string, []byte) {
	// shortcut for most scenarios
	if len(tmpl.partProviders) == 1 {
		return tmpl.partProviders[0](fields), buffer
	}
	buf := buffer[:0]
	for _, provide := range tmpl.partProviders {
		part := provide(fields)
		buf = append(buf, part...)
	}
	return util.DeepCopyStringFromBytes(buf), buf[:0]
}

func newPartResolverForString(s string) PartProvider {
	return func(source RecordType) string {
		return s
	}
}

func createVariableExpressionSolver(variableResolver PartProvider, expressionSubmatches []string) PartProvider {
	paramStartStr := expressionSubmatches[capturedStartIndex]
	paramEndStr := expressionSubmatches[capturedEndIndex]
	var err error
	paramStart := 0
	paramEnd := math.MaxInt32
	if paramStartStr != "" {
		paramStart, err = strconv.Atoi(paramStartStr)
		if err != nil {
			panic(err)
		}
	}
	if paramEndStr != "" {
		paramEnd, err = strconv.Atoi(paramEndStr)
		if err != nil {
			panic(err)
		}
	}
	return func(source RecordType) string {
		v := variableResolver(source)
		start := paramStart
		if start < 0 {
			// e.g. "-2:" of [abc] => [bc]
			start += len(v)
		}
		if start < 0 {
			// e.g. "-5:" of [abc] => [abc]
			start = 0
		}
		if start >= len(v) {
			return ""
		}
		end := paramEnd
		if end < 0 {
			// e.g. ":-1" of [abc] => [ab]
			end += len(v)
		}
		if end < 0 {
			// e.g. ":-5" of [abc] => []
			return ""
		}
		if end > len(v) {
			end = len(v)
		}
		if start < end {
			return v[start:end]
		}
		return ""
	}
}
