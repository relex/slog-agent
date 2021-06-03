package bmatch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gobwas/glob"
	"github.com/relex/slog-agent/util"
	"gopkg.in/yaml.v3"
)

// valueMatch configures how to match the value of one field
// for example: equality to "apache-tomcat", without the field name itself
type valueMatch struct {
	match       valueMatcher
	description string
	cost        int // relative cost used to decide which match should be checked first
}

var valueMatcherConstructors = map[string]valueMatcherConstructor{
	"!!str":         createValueMatcherStringEqualsTo, // the default tag for string value
	"!!str-any":     createValueMatcherStringAny,
	"!!str-eq":      createValueMatcherStringEqualsTo,
	"!!str-not":     createValueMatcherStringNotEqualsTo,
	"!!str-start":   createValueMatcherStringStartsWith,
	"!!str-end":     createValueMatcherStringEndsWith,
	"!!str-contain": createValueMatcherStringContains,
	"!!glob":        createValueMatcherGlob,
	"!!regex":       createValueMatcherRegex,
	"!!len-gt":      createValueMatcherLengthGreaterThan,
	"!!len-lt":      createValueMatcherLengthLessThan,
}

type valueMatcherConstructor = func(expression string) (valueMatch, error)

type valueMatcher = func(value string) bool

var emptyValueMatch = valueMatch{}

// MarshalYAML provides custom marshalling to export readable document. The result is not reversible.
func (match valueMatch) MarshalYAML() (interface{}, error) {
	return match.description, nil
}

func (match valueMatch) String() string {
	return match.description
}

func (match *valueMatch) UnmarshalYAML(value *yaml.Node) error {
	if creator, found := valueMatcherConstructors[value.Tag]; found {
		m, err := creator(value.Value)
		if err != nil {
			return util.NewYamlError(value, fmt.Sprintf("Failed value-match of tag %s: %s", value.Tag, err.Error()))
		}
		*match = m
	} else {
		return util.NewYamlError(value, fmt.Sprintf("Unsupported value-match tag: %s", value.Tag))
	}
	return nil
}

func createValueMatcherStringAny(expr string) (valueMatch, error) {
	if expr != "" {
		return emptyValueMatch, fmt.Errorf("value must be empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return len(v) > 0
		},
		description: "not-nil",
		cost:        0,
	}, nil
}

func createValueMatcherStringEqualsTo(expr string) (valueMatch, error) {
	if expr == "" {
		return emptyValueMatch, fmt.Errorf("value is empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return v == expr
		},
		description: "== " + expr,
		cost:        1 + len(expr)/2,
	}, nil
}

func createValueMatcherStringNotEqualsTo(expr string) (valueMatch, error) {
	if expr == "" {
		return emptyValueMatch, fmt.Errorf("value is empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return v != expr
		},
		description: "!= " + expr,
		cost:        1 + len(expr)/2,
	}, nil
}

func createValueMatcherStringStartsWith(expr string) (valueMatch, error) {
	if expr == "" {
		return emptyValueMatch, fmt.Errorf("value is empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return strings.HasPrefix(v, expr)
		},
		description: "ˆ= " + expr,
		cost:        1 + len(expr)/2,
	}, nil
}

func createValueMatcherStringEndsWith(expr string) (valueMatch, error) {
	if expr == "" {
		return emptyValueMatch, fmt.Errorf("value is empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return strings.HasSuffix(v, expr)
		},
		description: "$= " + expr,
		cost:        1 + len(expr)/2,
	}, nil
}

func createValueMatcherStringContains(expr string) (valueMatch, error) {
	if expr == "" {
		return emptyValueMatch, fmt.Errorf("value is empty")
	}
	return valueMatch{
		match: func(v string) bool {
			return strings.Contains(v, expr)
		},
		description: "*= " + expr,
		cost:        500 + len(expr),
	}, nil
}

func createValueMatcherGlob(expr string) (valueMatch, error) {
	g, err := glob.Compile(expr)
	if err != nil {
		return emptyValueMatch, err
	}
	return valueMatch{
		match:       g.Match,
		description: "~= " + expr,
		cost:        2000 + len(expr),
	}, nil
}

func createValueMatcherRegex(expr string) (valueMatch, error) {
	regex, err := regexp.Compile(expr)
	if err != nil {
		return emptyValueMatch, err
	}
	return valueMatch{
		match:       regex.MatchString,
		description: "~= " + expr,
		cost:        20000 + len(expr),
	}, nil
}

func createValueMatcherLengthGreaterThan(expr string) (valueMatch, error) {
	target, err := strconv.Atoi(expr)
	if err != nil {
		return emptyValueMatch, err
	}
	return valueMatch{
		match: func(v string) bool {
			return len(v) > target
		},
		description: "len > " + expr,
		cost:        0,
	}, nil
}

func createValueMatcherLengthLessThan(expr string) (valueMatch, error) {
	target, err := strconv.Atoi(expr)
	if err != nil {
		return emptyValueMatch, err
	}
	return valueMatch{
		match: func(v string) bool {
			return len(v) < target
		},
		description: "len < " + expr,
		cost:        0,
	}, nil
}
