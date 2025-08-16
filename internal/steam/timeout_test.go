package steam

import (
	"os"
	"testing"
	"time"
)

func TestTimeoutConfiguration(t *testing.T) {
	defer os.Unsetenv("ACHIEVEMENTS_TIMEOUT_SECS")

	t.Run("Default timeout", func(t *testing.T) {
		os.Unsetenv("ACHIEVEMENTS_TIMEOUT_SECS")
		if timeout := achievementTimeout(); timeout != 5*time.Second {
			t.Errorf("Expected 5s default, got %v", timeout)
		}
	})

	t.Run("Valid environment variable", func(t *testing.T) {
		os.Setenv("ACHIEVEMENTS_TIMEOUT_SECS", "10")
		if timeout := achievementTimeout(); timeout != 10*time.Second {
			t.Errorf("Expected 10s, got %v", timeout)
		}
	})

	t.Run("Invalid environment variable", func(t *testing.T) {
		os.Setenv("ACHIEVEMENTS_TIMEOUT_SECS", "invalid")
		if timeout := achievementTimeout(); timeout != 5*time.Second {
			t.Errorf("Expected 5s fallback, got %v", timeout)
		}
	})
}
