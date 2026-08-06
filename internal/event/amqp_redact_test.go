package event

import "testing"

// The startup line used to log cfg.URL verbatim, which for RABBITMQ_URL means
// "amqp://user:password@host" — a live broker credential written into the
// application log on every boot.
func TestRedactAMQPURLRemovesThePassword(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"user and password", "amqp://devuser:sup3rsecret@rabbitmq:5672/", "amqp://redacted@rabbitmq:5672/"},
		{"user only", "amqp://devuser@rabbitmq:5672/", "amqp://redacted@rabbitmq:5672/"},
		{"no userinfo", "amqp://rabbitmq:5672/", "amqp://rabbitmq:5672/"},
		{"vhost preserved", "amqps://u:p@host:5671/prod", "amqps://redacted@host:5671/prod"},
		{"unparseable is not echoed", "amqp://u:p@[::1:bad", "amqp://[unparseable]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactAMQPURL(tt.in)
			if got != tt.want {
				t.Fatalf("redactAMQPURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRedactAMQPURLNeverLeaksTheSecretSubstring(t *testing.T) {
	const password = "sup3rsecret"
	got := redactAMQPURL("amqp://devuser:" + password + "@rabbitmq:5672/")
	if contains(got, password) {
		t.Fatalf("redacted URL still contains the password: %q", got)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}()
}
