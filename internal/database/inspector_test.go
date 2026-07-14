package database

import "testing"

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   int64
		want string
	}{
		{name: "bytes", in: 512, want: "512 B"},
		{name: "kilobytes", in: 2 * 1024, want: "2.0 KB"},
		{name: "megabytes", in: 5 * 1024 * 1024, want: "5.0 MB"},
		{name: "gigabytes", in: 3 * 1024 * 1024 * 1024, want: "3.0 GB"},
		{name: "terabytes", in: 2 * 1024 * 1024 * 1024 * 1024, want: "2.0 TB"},
		{name: "petabyte reported in TB", in: 1 << 50, want: "1024.0 TB"},
		{name: "multi petabyte", in: 5 << 50, want: "5120.0 TB"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatBytes(tc.in); got != tc.want {
				t.Fatalf("formatBytes(%d) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
