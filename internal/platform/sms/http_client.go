package sms

import (
	"net/http"
	"time"
)

// httpClient is the shared client for outbound SMS-provider API calls (Twilio,
// Vonage). It carries an explicit client-level timeout so a stalled provider
// cannot pin a goroutine indefinitely even when the caller passes a context
// without a deadline. http.DefaultClient has no timeout of its own, which is why
// these callers must not use it.
var httpClient = &http.Client{Timeout: 15 * time.Second}
