package ids

import "testing"

func TestProjectDisplayID(t *testing.T) {
	tests := []struct {
		seq  uint64
		want string
	}{
		{seq: 1, want: "PRJ-0001"},
		{seq: 42, want: "PRJ-0042"},
		{seq: 9999, want: "PRJ-9999"},
		{seq: 10000, want: "PRJ-10000"},
	}

	for _, tt := range tests {
		if got := ProjectDisplayID(tt.seq); got != tt.want {
			t.Fatalf("ProjectDisplayID(%d) = %q, want %q", tt.seq, got, tt.want)
		}
	}
}
