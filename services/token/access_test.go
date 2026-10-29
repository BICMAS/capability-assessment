package token

import "testing"

func TestRoleProfilePermissions(t *testing.T) {
	cases := []struct {
		role     string
		maintain bool
		approve  bool
		read     bool
		orgWrite bool
		orgRead  bool
	}{
		{RoleAdministrator, true, true, true, true, true},
		{RoleHRLD, true, false, true, true, true},
		{RoleBusinessLeader, false, true, true, false, true},
		{RoleLineManager, false, false, true, false, true},
		{RoleEmployee, false, false, false, false, false},
	}
	for _, tc := range cases {
		claims := Claims{Roles: []string{tc.role}}
		if claims.CanMaintainRoleProfiles() != tc.maintain || claims.CanApproveRoleProfiles() != tc.approve || claims.CanReadRoleProfiles() != tc.read {
			t.Fatalf("%s profile permissions = maintain %v approve %v read %v", tc.role, claims.CanMaintainRoleProfiles(), claims.CanApproveRoleProfiles(), claims.CanReadRoleProfiles())
		}
		if claims.CanMaintainOrganogram() != tc.orgWrite || claims.CanReadOrganogram() != tc.orgRead {
			t.Fatalf("%s organogram permissions = write %v read %v", tc.role, claims.CanMaintainOrganogram(), claims.CanReadOrganogram())
		}
	}
}
