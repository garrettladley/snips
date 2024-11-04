package watcher

import "testing"

func TestShouldIncludeFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "basic true",
			path: "snippet_0.code.go",
			want: true,
		},
		{
			name: "basic false",
			path: "snippet_0.go",
			want: false,
		},
		{
			name: "multiple \".\"'s true",
			path: "foo.bar.code.rs",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIncludeFile(tt.path); got != tt.want {
				t.Errorf("shouldIncludeFile(\"%s\") = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
