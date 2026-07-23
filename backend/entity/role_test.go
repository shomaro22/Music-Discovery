package entity

import "testing"

func TestRoleValidation(t *testing.T) {
	if !RoleUser.IsPersistable() || !RoleContentAdmin.IsPersistable() || !RoleSystemAdmin.IsPersistable() {
		t.Fatal("persistable roles must be valid")
	}
	if RoleGuest.IsPersistable() || Role("USER").IsValid() || Role("unknown").IsValid() {
		t.Fatal("guest, case variants, and unknown roles must not be persistable")
	}
	if IsAllowedRole(RoleSystemAdmin, RoleContentAdmin) {
		t.Fatal("system admin must not implicitly include content admin")
	}
}
