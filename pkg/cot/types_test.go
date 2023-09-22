package cot

import (
	"fmt"
	"testing"
)

func TestGetType(t *testing.T) {
	tp := types["A-M-H"]

	if len(GetNext(tp.Code)) != 14 {
		t.Errorf("has %d next", len(GetNext(tp.Code)))
	}

	for _, s := range types["G-U-i"].Next {
		fmt.Println(s.Code, s.Name)
	}
}
