package userctx

import "context"

// Context key type
type contextKey string

const userEmailKey contextKey = "user_email"
const UserIDKey contextKey = "user_id"

// SetUserEmail adds user email to request context
func SetUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userEmailKey, email)
}

// GetUserEmail retrieves user email from request context
func GetUserEmail(ctx context.Context) string {
	email, ok := ctx.Value(userEmailKey).(string)
	if !ok {
		return "anonymous"
	}
	return email
}

// SetUserID adds user ID to request context
func SetUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, UserIDKey, id)
}

// GetUserID retrieves user ID from request context
func GetUserID(ctx context.Context) string {
	if userID := ctx.Value(UserIDKey); userID != nil {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}
