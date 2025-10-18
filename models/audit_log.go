package models

import "time"

// AuditLogEntry represents a single HTTP mutation event
type AuditLogEntry struct {
	ID        int64
	Timestamp time.Time
	UserEmail string
	Method    string
	Path      string
	FormData  string
	UserAgent string
	IPAddress string
}
