package openurl

import (
	"reflect"
	"testing"
)

func TestCommand(t *testing.T) {
	cases := []struct {
		goos, name string
		args       []string
	}{
		{"linux", "xdg-open", []string{"https://x.test"}},
		{"darwin", "open", []string{"https://x.test"}},
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", "https://x.test"}},
	}
	for _, tc := range cases {
		name, args := Command(tc.goos, "https://x.test")
		if name != tc.name || !reflect.DeepEqual(args, tc.args) {
			t.Fatalf("%s: got %q %v want %q %v", tc.goos, name, args, tc.name, tc.args)
		}
	}
}
