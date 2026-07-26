package security

import "os"

// devLogOTP is true only when OTP echoing is explicitly enabled for LOCAL
// development via MAINTAINERD_DEV_LOG_OTP=true. It MUST remain false in every
// deployed environment: an OTP is a single-use credential, and logging it (logs
// reach stdout, aggregators, and screen shares) lets anyone with log access
// complete an SMS login or satisfy an MFA step-up within the code's TTL.
var devLogOTP = os.Getenv("MAINTAINERD_DEV_LOG_OTP") == "true"

// RedactedOTP returns the OTP for logging ONLY when dev OTP echo is explicitly
// enabled; otherwise it returns a redaction marker. Every OTP delivery-failure
// log MUST pass its code through this so credential material never lands in logs
// in a deployed environment.
func RedactedOTP(code string) string {
	if devLogOTP {
		return code
	}
	return "[redacted]"
}
