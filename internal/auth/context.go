
package auth

import "context"

type contextKey string

const(
    userIDKey contextKey = "userID"
    userRoleKey contextKey = "user_role"
)
func SetUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok && userID != ""
}

func SetUserRole(ctx context.Context, role string) context.Context {
    return context.WithValue(ctx, userRoleKey, role)
}

func GetUserRole(ctx context.Context) string {
    role, _ := ctx.Value(userRoleKey).(string)
    return role
}
