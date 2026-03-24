package models

type Team struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	CreatedBy uint64 `json:"created_by"`
}

type TeamMemberRole string

const (
	RoleOwner  TeamMemberRole = "owner"
	RoleAdmin  TeamMemberRole = "admin"
	RoleMember TeamMemberRole = "member"
)
