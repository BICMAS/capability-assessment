package token

// Role names carried on the internal token. An enterprise identity provider
// maps its groups onto these values. Services authorise from the token, not
// from a role stored beside the domain record.
const (
	RoleAdministrator  = "admin"
	RoleHRLD           = "hr"
	RoleLineManager    = "manager"
	RoleBusinessLeader = "leader"
	RoleEmployee       = "employee"
)

// CanMaintainRoleProfiles is the authoring permission for administrators and
// HR/L&D specialists. They prepare drafts, sources, clarification, and generation.
func (c Claims) CanMaintainRoleProfiles() bool {
	return c.HasRole(RoleAdministrator) || c.HasRole(RoleHRLD)
}

// CanApproveRoleProfiles is the approval permission. Administrators and
// business leaders publish a profile that is in review.
func (c Claims) CanApproveRoleProfiles() bool {
	return c.HasRole(RoleAdministrator) || c.HasRole(RoleBusinessLeader)
}

// CanReadRoleProfiles allows authoring, approval, and line-manager reads.
// Employees do not read draft authoring records.
func (c Claims) CanReadRoleProfiles() bool {
	return c.CanMaintainRoleProfiles() || c.CanApproveRoleProfiles() || c.HasRole(RoleLineManager)
}

// CanMaintainOrganogram allows administrators and HR/L&D to record positions.
func (c Claims) CanMaintainOrganogram() bool {
	return c.HasRole(RoleAdministrator) || c.HasRole(RoleHRLD)
}

// CanReadOrganogram allows the people who staff or approve roles to read the chart.
func (c Claims) CanReadOrganogram() bool {
	return c.CanMaintainOrganogram() || c.HasRole(RoleBusinessLeader) || c.HasRole(RoleLineManager)
}
