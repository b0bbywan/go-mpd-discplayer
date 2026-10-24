package retry

import (
	"errors"
	"testing"
	"time"
)

func TestUntilFirstTry(t *testing.T) {
	calls := 0
	err := Until(time.Second, time.Hour, func() error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Errorf("Until() = %v after %d calls, want nil after 1", err, calls)
	}
}

func TestUntilRetriesUntilSuccess(t *testing.T) {
	calls := 0
	err := Until(time.Second, time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("Until() = %v after %d calls, want nil after 3", err, calls)
	}
}

func TestUntilTimeout(t *testing.T) {
	calls := 0
	start := time.Now()
	err := Until(50*time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		return errors.New("still failing")
	})
	if err == nil || err.Error() != "still failing" {
		t.Errorf("Until() error = %v, want the last error", err)
	}
	if calls < 2 {
		t.Errorf("fn called %d times, want several", calls)
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond || elapsed > time.Second {
		t.Errorf("Until() returned after %s, want about 50ms", elapsed)
	}
}
