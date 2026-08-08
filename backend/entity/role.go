package entity

type Role string

const (
	RoleGuest        Role = "guest"
	RoleUser         Role = "user"
	RoleContentAdmin Role = "content_admin"
	RoleSystemAdmin  Role = "system_admin"
)

func (r Role) IsValid() bool {
	return r == RoleGuest || r.IsPersistable()
}

func (r Role) IsPersistable() bool {
	return r == RoleUser || r == RoleContentAdmin || r == RoleSystemAdmin
}

func (r Role) IsAdmin() bool        { return r == RoleContentAdmin || r == RoleSystemAdmin }
func (r Role) IsContentAdmin() bool { return r == RoleContentAdmin }
func (r Role) IsSystemAdmin() bool  { return r == RoleSystemAdmin }

func IsAllowedRole(current Role, allowed ...Role) bool {
	for _, role := range allowed {
		if current == role {
			return true
		}
	}
	return false
}
