package btest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubLogCustomCounter(t *testing.T) {
	reg, lookup := NewStubLogCustomCounterRegistry()
	a := reg.RegisterCustomCounter("a")
	b := reg.RegisterCustomCounter("b")
	b2 := reg.RegisterCustomCounter("b")

	a(3)
	b(7)
	b2(11)

	acnt, alen := lookup("a")
	assert.Equal(t, int64(1), acnt)
	assert.Equal(t, int64(3), alen)

	bcnt, blen := lookup("b")
	assert.Equal(t, int64(2), bcnt)
	assert.Equal(t, int64(18), blen)
}
