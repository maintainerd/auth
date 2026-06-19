package dashboard

type ResourceCount struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive,omitempty"`
}

type UserCount struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Inactive  int64 `json:"inactive"`
	Suspended int64 `json:"suspended"`
	Pending   int64 `json:"pending"`
}

type SummaryResponse struct {
	Users              UserCount     `json:"users"`
	Services           ResourceCount `json:"services"`
	Clients            ResourceCount `json:"clients"`
	IdentityProviders  ResourceCount `json:"identity_providers"`
	Roles              ResourceCount `json:"roles"`
	APIKeys            ResourceCount `json:"api_keys"`
	RecentLogins24h    int64         `json:"recent_logins_24h"`
	FailedLogins24h    int64         `json:"failed_logins_24h"`
}
