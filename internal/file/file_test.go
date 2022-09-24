package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceName(t *testing.T) {
	tests := []struct {
		deviceNames map[string][]string
		originalFilePath string
		expect string
	}{
		{
			map[string][]string{"alice": {"phone"}},
			"/a/phone",
			"alice",
		},
		{
			map[string][]string{"alice": {"/a/phone"}},
			"/a/phone",
			"alice",
		},
		{
			map[string][]string{"alice": {"/a/phone"}, "bob": {"/b/phone"}},
			"/a/phone/b",
			"alice",
		},
		{
			map[string][]string{"alice": {"/a/phone"}},
			"/foo/bar/baz",
			"",
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, test.expect, DeviceName(test.deviceNames, test.originalFilePath))
		})
	}
}
