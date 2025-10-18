package repositories

import (
	"database/sql"
	"time"

	"github.com/blogem/eod-scheduler/models"
)

// AuditRepository handles audit log persistence
type AuditRepository interface {
	Create(entry *models.AuditLogEntry) error
}

type sqliteAuditRepository struct {
	db *sql.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *sql.DB) AuditRepository {
	return &sqliteAuditRepository{db: db}
}

// Create inserts a new audit log entry
func (r *sqliteAuditRepository) Create(entry *models.AuditLogEntry) error {
	query := `
		INSERT INTO audit_log (timestamp, user_email, method, path, form_data, user_agent, ip_address)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		time.Now(),
		entry.UserEmail,
		entry.Method,
		entry.Path,
		entry.FormData,
		entry.UserAgent,
		entry.IPAddress,
	)

	return err
}
