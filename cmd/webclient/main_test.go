package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeUid(t *testing.T) {
	c := makeUID("s1")
	fmt.Println(c)
	assert.Len(t, c, 24)
	assert.Equal(t, makeUID("string1"), makeUID("string1"))
	assert.NotEqual(t, makeUID("s1"), makeUID("s2"))
}
