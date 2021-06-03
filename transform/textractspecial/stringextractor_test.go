package textractspecial

import (
	"testing"

	"github.com/relex/slog-agent/util"
	"github.com/stretchr/testify/assert"
)

func TestExtractor(t *testing.T) {
	if ex, err := newStringExtractor(extractFromStart, []string{"[", "*", "]"}, 100); assert.Nil(t, err) {
		lbl, txt := ex.Extract("[Hello]Message")
		assert.Equal(t, "Hello", lbl)
		assert.Equal(t, "Message", txt)
	}
	if ex, err := newStringExtractor(extractFromEnd, []string{"", "[^ ]", ""}, 100); assert.Nil(t, err) {
		lbl, txt := ex.Extract("Lorem ipsum dolor sit amet, consectetur adipiscing elit")
		assert.Equal(t, "elit", lbl)
		assert.Equal(t, "Lorem ipsum dolor sit amet, consectetur adipiscing ", txt)
	}
	if ex, err := newStringExtractor(extractFromEnd, []string{".", "[0-9]", ""}, 100); assert.Nil(t, err) {
		{
			num, fn := ex.Extract("error.log.123")
			assert.Equal(t, "123", num)
			assert.Equal(t, "error.log", fn)
		}
		{
			num, fn := ex.Extract("error.log.bz2")
			assert.Equal(t, "", num)
			assert.Equal(t, "error.log.bz2", fn)
		}
	}
	if ex, err := newStringExtractorSimple(extractFromStart, `([0-9a-z\]])`, 100); assert.Nil(t, err) {
		{
			lbl, txt := ex.Extract("(x12]3)Foo")
			assert.Equal(t, "x12]3", lbl)
			assert.Equal(t, "Foo", txt)
		}
		{
			lbl, txt := ex.Extract("(X12]3)Foo")
			assert.Equal(t, "", lbl)
			assert.Equal(t, "(X12]3)Foo", txt)
		}
	}
}

func TestExtractorCompile(t *testing.T) {
	{
		parts, err := splitPattern(`\[*\] - `)
		assert.Nil(t, err)
		assert.Equal(t, []string{"[", "*", "] - "}, parts)
	}
	{
		parts, err := splitPattern(`Foo\[Bar[0-9\]]:`)
		assert.Nil(t, err)
		assert.Equal(t, []string{"Foo[Bar", `[0-9\]]`, ":"}, parts)
	}
	{
		tbl := make([]bool, 256)
		assert.Nil(t, fillValidCharsByRangeExpression(tbl, "[a-z0-9]"))
		assert.False(t, tbl[' '])
		assert.False(t, tbl['0'-1])
		assert.False(t, tbl['9'+1])
		assert.False(t, tbl['a'-1])
		assert.False(t, tbl['z'+1])
		assert.True(t, tbl['0'])
		assert.True(t, tbl['9'])
		assert.True(t, tbl['a'])
		assert.True(t, tbl['m'])
		assert.True(t, tbl['z'])
	}
	{
		tbl := make([]bool, 256)
		assert.Nil(t, fillValidCharsByRangeExpression(tbl, "[^A-Zxmz-]"))
		assert.True(t, tbl[' '])
		assert.True(t, tbl['A'-1])
		assert.True(t, tbl['Z'+1])
		assert.True(t, tbl['a'])
		assert.True(t, tbl['n'])
		assert.False(t, tbl['A'])
		assert.False(t, tbl['B'])
		assert.False(t, tbl['Z'])
		assert.False(t, tbl['x'])
		assert.False(t, tbl['m'])
		assert.False(t, tbl['z'])
		assert.False(t, tbl['-'])
	}
	{
		tbl := make([]bool, 256)
		assert.Error(t, fillValidCharsByRangeExpression(tbl, "[^abc--x]"), "double hyphen at index 6")
	}
}

func TestExtractLabelAtStartWithBoundaries(t *testing.T) {
	msg := `[GroupLoader              ] - Hello World [OPEN] - Yes`
	{
		comp, log := extractLabelAtStart(msg, "[", "] - ", 50, nil)
		assert.Equal(t, "GroupLoader", comp)
		assert.Equal(t, "Hello World [OPEN] - Yes", log)
	}
	{
		comp, log := extractLabelAtStart(msg, "[", "] - ", 20, nil)
		assert.Equal(t, "", comp)
		assert.Equal(t, msg, log)
	}
}

func TestExtractLabelAtStartWithRightBoundary(t *testing.T) {
	msg := `name=Foo, Hello World [OPEN]`
	{
		comp, log := extractLabelAtStart(msg, "", ", ", 50, nil)
		assert.Equal(t, "name=Foo", comp)
		assert.Equal(t, "Hello World [OPEN]", log)
	}
	{
		tbl := make([]bool, 256)
		util.Each(len(tbl), func(c int) { tbl[c] = true })
		tbl['='] = false
		comp, log := extractLabelAtStart(msg, "", ", ", 50, tbl)
		assert.Equal(t, "", comp)
		assert.Equal(t, msg, log)
	}
}

func TestExtractLabelAtStartWithoutRightBoundary(t *testing.T) {
	msg := `id=012345 Hello World [OPEN]`
	{
		tbl := make([]bool, 256)
		for c := '0'; c <= '9'; c++ {
			tbl[c] = true
		}
		for c := 'a'; c <= 'z'; c++ {
			tbl[c] = true
		}
		tbl['='] = true
		comp, log := extractLabelAtStart(msg, "", "", 50, tbl)
		assert.Equal(t, "id=012345", comp)
		assert.Equal(t, " Hello World [OPEN]", log)
	}
}

func TestExtractLabelAtEndWithBoundaries(t *testing.T) {
	msg := `{x} Hello World Message 123456789 <{ Foo }`
	{
		comp, log := extractLabelAtEnd(msg, "<{", "}", 50, nil)
		assert.Equal(t, "Foo", comp)
		assert.Equal(t, "{x} Hello World Message 123456789 ", log)
	}
	{
		comp, log := extractLabelAtEnd(msg, "<{", "}", 10, nil)
		assert.Equal(t, "Foo", comp)
		assert.Equal(t, "{x} Hello World Message 123456789 ", log)
	}
	{
		comp, log := extractLabelAtEnd(msg, "<{", "}", 5, nil)
		assert.Equal(t, "", comp)
		assert.Equal(t, msg, log)
	}
}

func TestExtractLabelAtEndWithRightBoundary(t *testing.T) {
	msg := `Hello World, name=Foo`
	{
		comp, log := extractLabelAtEnd(msg, ", ", "", 50, nil)
		assert.Equal(t, "name=Foo", comp)
		assert.Equal(t, "Hello World", log)
	}
	{
		tbl := make([]bool, 256)
		util.Each(len(tbl), func(c int) { tbl[c] = true })
		tbl['='] = false
		comp, log := extractLabelAtEnd(msg, "", ", ", 50, tbl)
		assert.Equal(t, "", comp)
		assert.Equal(t, msg, log)
	}
}

func TestExtractLabelAtEndWithoutRightBoundary(t *testing.T) {
	msg := `Filename012345$`
	{
		tbl := make([]bool, 256)
		for c := '0'; c <= '9'; c++ {
			tbl[c] = true
		}
		comp, log := extractLabelAtEnd(msg, "", "$", 50, tbl)
		assert.Equal(t, "012345", comp)
		assert.Equal(t, "Filename", log)
	}
}
