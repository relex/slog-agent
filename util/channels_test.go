package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollectFromChannel(t *testing.T) {
	c := make(chan int, 10)
	c <- 1
	c <- 2
	c <- 3
	close(c)

	assert.EqualValues(t, []int{1, 2, 3}, CollectFromChannel(c))
}
