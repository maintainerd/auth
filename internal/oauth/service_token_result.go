package oauth

// OAuthTokenResult is the internal result produced by the token service and
// rendered to the wire by the token handler.
type OAuthTokenResult struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    int64
	RefreshToken string
	IDToken      string
	Scope        string
}
