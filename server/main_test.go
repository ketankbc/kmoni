package main

import "testing"

func TestSame(t *testing.T) {
	if !same("abc", "abc") || same("abc", "abd") || same("abc", "abcd") {
		t.Fatal("constant comparison failed")
	}
}
