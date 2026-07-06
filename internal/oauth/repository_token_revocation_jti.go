package oauth

// IsRevokedByJTI checks whether a JTI has been revoked without scoping to a
// specific tenant. A JTI is globally unique per RFC 7519 §4.1.7, so a
// cross-tenant lookup is correct here. Used by the JWT bearer middleware where
// tenant context is not available before claims are fully decoded.
func (r *oauthTokenRevocationRepository) IsRevokedByJTI(jti string) (bool, error) {
	var count int64
	err := r.DB().Model(&OAuthTokenRevocation{}).
		Where("jti = ?", jti).
		Count(&count).Error
	return count > 0, err
}
