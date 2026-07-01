package accesslog

// Event is a short-lived diagnostic record for recent node access.
// It is reported to the panel for troubleshooting and should not be used as
// a long-term audit log.
type Event struct {
	UserID      int    `json:"user_id"`
	XrayEmail   string `json:"xray_email"`
	Source      string `json:"source"`
	Network     string `json:"network"`
	Destination string `json:"destination"`
	Timestamp   int64  `json:"timestamp"`
}
