package textractspecial

import (
	"fmt"
	"strings"

	"github.com/relex/slog-agent/util/stringunescape"
)

type stringExtractorPosition int

const (
	extractFromStart stringExtractorPosition = 1
	extractFromEnd   stringExtractorPosition = 2
)

var patternUnescaper = stringunescape.NewUnescaper('\\', map[byte]byte{
	'[': '[',
	']': ']',
	'*': '*',
})

// stringExtractor is a fast alternative to string extraction by regular expression, using simple boundaries and table of valid bytes
type stringExtractor struct {
	position   stringExtractorPosition
	leftBound  string
	rightBound string
	maxRange   int
	validChars []bool
}

var emptyExtractor = stringExtractor{}

// newStringExtractorSimple creates a StringExtractor or returns error if the given pattern is not supported
// A pattern is like "Foo*Bar" or "component=[a-z0-9_],", where there has to be exactly one wildcard or brackets
func newStringExtractorSimple(position stringExtractorPosition, pattern string, maxRange int) (stringExtractor, error) {
	parts, err := splitPattern(pattern)
	if err != nil {
		return emptyExtractor, err
	}
	return newStringExtractor(position, parts, maxRange)
}

// newStringExtractor creates a StringExtractor or returns error if the given pattern parts are not supported
// The only type of patterns supported is ["left boundary", "wildcard for the target", "right boundary"]
// Both boundaries are optional and may be empty, while the target wildcard can be either "*" or "[a-z123...A-Z-]", but not both or combined.
func newStringExtractor(position stringExtractorPosition, patternParts []string, maxRange int) (stringExtractor, error) {
	if len(patternParts) != 3 {
		return emptyExtractor, fmt.Errorf("must have 3 parts: [left, wildcard, right], not %d", len(patternParts))
	}
	leftBoundary := patternParts[0]
	targetWildcard := patternParts[1]
	rightBoundary := patternParts[2]
	var validCharTable []bool
	switch {
	case len(targetWildcard) == 0:
		return emptyExtractor, fmt.Errorf("patternParts[1] must not be empty")
	case targetWildcard == "*":
		validCharTable = nil
	case len(targetWildcard) < 2 || targetWildcard[0] != '[' || targetWildcard[len(targetWildcard)-1] != ']':
		return emptyExtractor, fmt.Errorf("patternParts[1] must be '*' or '[...]'")
	default:
		validCharTable = make([]bool, 256)
		if err := fillValidCharsByRangeExpression(validCharTable, targetWildcard); err != nil {
			return emptyExtractor, fmt.Errorf("patternParts[1]: %w", err)
		}
	}
	return stringExtractor{
		position:   position,
		leftBound:  leftBoundary,
		rightBound: rightBoundary,
		maxRange:   maxRange,
		validChars: validCharTable,
	}, nil
}

// Extract extracts label from the given text
// Returns (label, text after cutting label and boundaries)
func (ex *stringExtractor) Extract(text string) (string, string) {
	switch ex.position {
	case extractFromStart:
		return extractLabelAtStart(text, ex.leftBound, ex.rightBound, ex.maxRange, ex.validChars)
	case extractFromEnd:
		return extractLabelAtEnd(text, ex.leftBound, ex.rightBound, ex.maxRange, ex.validChars)
	default:
		panic(ex.position)
	}
}

// splitPattern splits pattern e.g. "\[*\] - " into "[", "*", "] - "
func splitPattern(pattern string) ([]string, error) {
	if i := patternUnescaper.FindFirstUnescaped(pattern, '*'); i != -1 {
		left := patternUnescaper.Run(pattern[:i])
		right := patternUnescaper.Run(pattern[i+1:])
		return []string{left, "*", right}, nil
	}
	if bstart := patternUnescaper.FindFirstUnescaped(pattern, '['); bstart != -1 {
		left := patternUnescaper.Run(pattern[:bstart])
		bend := patternUnescaper.FindFirstUnescaped(pattern[bstart+1:], ']')
		if bend == -1 {
			return nil, fmt.Errorf("bracket not closed")
		}
		bend += bstart + 1
		bracket := pattern[bstart : bend+1] // str inside bracket shouldn't be escaped
		right := patternUnescaper.Run(pattern[bend+1:])
		return []string{left, bracket, right}, nil
	}
	return nil, fmt.Errorf("no wildcard or bracket")
}

// fillValidCharsByRangeExpression fills the char table by bracket expression e.g. [a-z0-9-]
func fillValidCharsByRangeExpression(table []bool, expression string) error {
	if len(expression) == 0 {
		return nil
	}
	expr := patternUnescaper.Run(expression[1 : len(expression)-1])
	if len(expr) == 0 {
		return fmt.Errorf("empty expression")
	}
	offset := 1
	var listedValue bool
	if expr[0] == '^' {
		for i := 0; i < len(table); i++ {
			table[i] = true
		}
		listedValue = false
		expr = expr[1:]
		offset++
	} else {
		for i := 0; i < len(table); i++ {
			table[i] = false
		}
		listedValue = true
	}
	rangeStarted := false
	// [A-Zabcd0-9]
	for i := 0; i < len(expr); i++ {
		c := expr[i]
		if c == '-' {
			if rangeStarted {
				return fmt.Errorf("double hyphen at index %d", i+offset)
			}
			if i > 0 && i < len(expr)-1 {
				rangeStarted = true
			} else {
				table['-'] = listedValue
			}
		} else {
			if rangeStarted {
				for rc := expr[i-2]; rc <= c; rc++ {
					table[rc] = listedValue
				}
				rangeStarted = false
			} else {
				table[c] = listedValue
			}
		}
	}
	return nil
}

// extractLabelAtStart extracts for example "Foo" from "[Foo]...text" with given boundaries, search range and valid chars
// The leftBoundary is optional, while at least one of rightBoundary or validChars should be present
// Returns (label, text with the label and its boundaries removed)
// If matching fails, returns ("", text)
func extractLabelAtStart(text string, leftBoundary string, rightBoundary string, maxRange int, validChars []bool) (string, string) {
	s := text
	if len(leftBoundary) > 0 {
		if !strings.HasPrefix(text, leftBoundary) {
			return "", text
		}
		s = s[len(leftBoundary):]
	}
	// fail fast if table match on tag won't succeed
	if len(s) > 0 && validChars != nil && !validChars[s[0]] {
		return "", text
	}
	if len(rightBoundary) > 0 {
		var iend int
		if len(s) > maxRange {
			iend = strings.Index(s[:maxRange], rightBoundary)
		} else {
			iend = strings.Index(s, rightBoundary)
		}
		if iend == -1 {
			return "", text
		}
		tag := s[:iend]
		if validChars != nil && matchValidCharsFromStart(tag, validChars) != len(tag) {
			return "", text
		}
		return trimControlCharsAndSpaces(tag), s[iend+len(rightBoundary):]
	}
	tagEnd := matchValidCharsFromStart(s, validChars)
	if tagEnd == 0 {
		return "", text
	}
	return trimControlCharsAndSpaces(s[:tagEnd]), s[tagEnd:]
}

// extractLabelAtEnd extracts for example "Bar" from "text...[Bar]" with given boundaries, search range and valid chars
// The rightBoundary is optional, while at least one of leftBoundary or validChars should be present
// Returns (label, text with the label and its boundaries removed)
// If matching fails, returns ("", text)
func extractLabelAtEnd(text string, leftBoundary string, rightBoundary string, maxRange int, validChars []bool) (string, string) {
	s := text
	if len(rightBoundary) > 0 {
		if !strings.HasSuffix(text, rightBoundary) {
			return "", text
		}
		s = s[:len(s)-len(rightBoundary)]
	}
	// fail fast if table match on tag won't succeed
	if len(s) > 0 && validChars != nil && !validChars[s[len(s)-1]] {
		return "", text
	}
	if len(leftBoundary) > 0 {
		var iend int
		if len(s) > maxRange {
			off := len(s) - maxRange
			iend = strings.LastIndex(s[off:], leftBoundary)
			if iend != -1 {
				iend += off
			}
		} else {
			iend = strings.LastIndex(s, leftBoundary)
		}
		if iend == -1 {
			return "", text
		}
		tag := s[iend+len(leftBoundary):]
		if validChars != nil && matchValidCharsFromEnd(tag, validChars) != 0 {
			return "", text
		}
		return trimControlCharsAndSpaces(tag), s[:iend]
	}
	tagBeg := matchValidCharsFromEnd(s, validChars)
	if tagBeg == len(s) {
		return "", text
	}
	return trimControlCharsAndSpaces(s[tagBeg:]), s[:tagBeg]
}

// matchValidCharsFromStart returns the end of substring from start which matches the given validChars
func matchValidCharsFromStart(s string, validChars []bool) int {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !validChars[c] {
			return i
		}
	}
	return len(s)
}

// matchValidCharsFromEnd returns the beginning of substring from end which matches the given validChars
func matchValidCharsFromEnd(s string, validChars []bool) int {
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if !validChars[c] {
			return i + 1
		}
	}
	return 0
}

func trimControlCharsAndSpaces(s string) string {
	istart := 0
	for istart < len(s) {
		if s[istart] > ' ' {
			break
		}
		istart++
	}
	iend := len(s) - 1
	for iend >= 0 {
		if s[iend] > ' ' {
			break
		}
		iend--
	}
	return s[istart : iend+1]
}
