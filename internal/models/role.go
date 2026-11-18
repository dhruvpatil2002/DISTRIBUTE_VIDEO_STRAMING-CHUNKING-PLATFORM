package models

const (
    RoleUser     = "user"
    RoleStreamer = "streamer"
    RoleAdmin    = "admin"
)


func IsValidRole(role string) bool {
	switch role {
	case RoleUser, RoleAdmin:
		return true
	default:
		return false
	}
}