package core

import (
	"testing"
)

func TestIsBool(t *testing.T) {
	_, b := IsBool("true")

	if !b {
		t.Error(`IsBool("true") = false`)
	}
}
