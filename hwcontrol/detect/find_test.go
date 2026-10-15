package detect

import (
	"errors"
	"testing"
)

func TestPickDrive(t *testing.T) {
	tests := []struct {
		drives  []string
		want    string
		wantErr error
	}{
		{nil, "", ErrNoDisc},
		{[]string{"/dev/sr1"}, "/dev/sr1", nil},
		{[]string{"/dev/sr0", "/dev/sr1"}, "", ErrSeveralDiscs},
	}
	for _, tt := range tests {
		got, err := pickDrive("a disc", tt.drives)
		if got != tt.want || !errors.Is(err, tt.wantErr) {
			t.Errorf("pickDrive(%v) = %q, %v; want %q, %v", tt.drives, got, err, tt.want, tt.wantErr)
		}
	}
}
