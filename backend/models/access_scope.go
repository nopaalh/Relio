package models

// Membership checks are defensive preconditions, not authentication or proof
// of a native deal-account relationship. The adapter must verify that link.
func (s AccessScope) AllowsAccount(accountID string) bool {
	return containsScopeID(s.AllowedAccountIDs, accountID)
}
func (s AccessScope) AllowsDeal(dealID, accountID string) bool {
	return containsScopeID(s.AllowedDealIDs, dealID) && s.AllowsAccount(accountID)
}
func (s AccessScope) AllowsAnalogAccount(accountID string) bool {
	return containsScopeID(s.AllowedAnalogAccountIDs, accountID)
}
func containsScopeID(ids []string, target string) bool {
	if target == "" {
		return false
	}
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}
