package scroll

import (
	"fmt"
	"testing"
)

var _ = fmt.Printf // for testing

func TestAllowSetBytes(t *testing.T) {
	tests := []struct {
		inString string
		inAllow  AllowSet
		out      bool
	}{
		// 0 - no match
		{
			"hello0",
			NewAllowSetBytes(`0123456789`, 100),
			false,
		},
		// 1 - length (input length is one more than max)
		{
			"hello",
			NewAllowSetBytes(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`, 4),
			false,
		},
		// 2 - length (equal)
		{
			"hello",
			NewAllowSetBytes(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`, 5),
			true,
		},
		// 3 - length (input length is one less than max)
		{
			"hello",
			NewAllowSetBytes(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ`, 6),
			true,
		},
		// 5 - all good
		{
			"hello, world",
			NewAllowSetBytes(`abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ, `, 100),
			true,
		},
	}

	for i, tt := range tests {
		if g, w := tt.inAllow.IsSafe(tt.inString), tt.out; g != w {
			t.Errorf("Test(%v), Got IsSafe: %v, Want: %v", i, g, w)
		}
	}
}

func TestAllowSetStrings(t *testing.T) {
	tests := []struct {
		inString string
		inAllow  AllowSet
		out      bool
	}{
		// 0 - no match
		{
			"foo",
			NewAllowSetStrings([]string{`bar`}),
			false,
		},
		// 1 - empty
		{
			"foo",
			NewAllowSetStrings([]string{``}),
			false,
		},
		// 2 - one less
		{
			"foo",
			NewAllowSetStrings([]string{`fo`}),
			false,
		},
		// 3 - one more
		{
			"foo",
			NewAllowSetStrings([]string{`fooo`}),
			false,
		},
		// 4 - exact match
		{
			"foo",
			NewAllowSetStrings([]string{`foo`}),
			true,
		},
	}

	for i, tt := range tests {
		if g, w := tt.inAllow.IsSafe(tt.inString), tt.out; g != w {
			t.Errorf("Test(%v), Got IsSafe: %v, Want: %v", i, g, w)
		}
	}
}
