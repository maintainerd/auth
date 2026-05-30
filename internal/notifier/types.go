package notifier

import "time"

// EmailConfigResponseDTO is the JSON representation of an email config record.
type EmailConfigResponseDTO struct {
	EmailConfigID string    `json:"email_config_id"`
	Provider      string    `json:"provider"`
	Host          string    `json:"host"`
	Port          int       `json:"port"`
	Username      string    `json:"username"`
	FromAddress   string    `json:"from_address"`
	FromName      string    `json:"from_name"`
	ReplyTo       string    `json:"reply_to"`
	Encryption    string    `json:"encryption"`
	TestMode      bool      `json:"test_mode"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EmailConfigUpdateRequestDTO is the request body for updating email config.
type EmailConfigUpdateRequestDTO struct {
	Provider    string `json:"provider"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	FromName    string `json:"from_name"`
	ReplyTo     string `json:"reply_to"`
	Encryption  string `json:"encryption"`
	TestMode    *bool  `json:"test_mode"`
}

// SMSConfigResponseDTO is the JSON representation of an SMS config record.
type SMSConfigResponseDTO struct {
	SMSConfigID string    `json:"sms_config_id"`
	Provider    string    `json:"provider"`
	AccountSID  string    `json:"account_sid"`
	FromNumber  string    `json:"from_number"`
	SenderID    string    `json:"sender_id"`
	TestMode    bool      `json:"test_mode"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// SMSConfigUpdateRequestDTO is the request body for updating SMS config.
type SMSConfigUpdateRequestDTO struct {
	Provider   string `json:"provider"`
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	FromNumber string `json:"from_number"`
	SenderID   string `json:"sender_id"`
	TestMode   *bool  `json:"test_mode"`
}
