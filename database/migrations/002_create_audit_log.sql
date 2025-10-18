-- Create audit_log table to track all HTTP mutations
CREATE TABLE IF NOT EXISTS audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    user_email TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    form_data TEXT,
    user_agent TEXT,
    ip_address TEXT
);

-- Index for common queries
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_email ON audit_log(user_email);
CREATE INDEX IF NOT EXISTS idx_audit_log_path ON audit_log(path);
