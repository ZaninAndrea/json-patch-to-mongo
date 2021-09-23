package jsonpatchtomongo

import (
	"fmt"
	"testing"
)

func TestParsePatches(t *testing.T) {
	want := "map[$push:map[hello.0.hi:map[$each:[4 3 2 1] $position:5]] $set:map[hello.0.hi.num:4]]"

	patches := []byte(`[
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 1 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 2 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 3 },
  		{ "op": "add", "path": "/hello/0/hi/5", "value": 4 },
  		{ "op": "add", "path": "/hello/0/hi/num", "value": 4 }
	]`)
	val, _, err := ParsePatches(patches)
	valStr := fmt.Sprint(val)

	if err != nil || valStr != want {
		t.Errorf("Hello() = %q, want %q", valStr, want)
	}
}
