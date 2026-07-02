package accesslog

import "sync/atomic"

// Event is a short-lived diagnostic record for recent node access.
// It is reported to the panel for troubleshooting and should not be used as
// a long-term audit log.
type Event struct {
	SessionID   string `json:"session_id,omitempty"`
	UserID      int    `json:"user_id"`
	XrayEmail   string `json:"xray_email"`
	Source      string `json:"source"`
	Network     string `json:"network"`
	Destination string `json:"destination"`
	Timestamp   int64  `json:"timestamp"`
	Upload      int64  `json:"upload,omitempty"`
	Download    int64  `json:"download,omitempty"`
}

// Activity tracks traffic for one short-lived access target.
type Activity struct {
	base Event

	upload   atomic.Int64
	download atomic.Int64

	lastUpload   atomic.Int64
	lastDownload atomic.Int64
	closed       atomic.Bool
}

func NewActivity(base Event) *Activity {
	a := &Activity{base: base}
	a.lastUpload.Store(-1)
	a.lastDownload.Store(-1)
	return a
}

func (a *Activity) AddUpload(n int64) {
	if a != nil && n > 0 {
		a.upload.Add(n)
	}
}

func (a *Activity) AddDownload(n int64) {
	if a != nil && n > 0 {
		a.download.Add(n)
	}
}

func (a *Activity) Close() {
	if a != nil {
		a.closed.Store(true)
	}
}

func (a *Activity) Closed() bool {
	return a == nil || a.closed.Load()
}

func (a *Activity) SnapshotIfChanged() (Event, bool) {
	if a == nil {
		return Event{}, false
	}
	upload := a.upload.Load()
	download := a.download.Load()
	lastUpload := a.lastUpload.Swap(upload)
	lastDownload := a.lastDownload.Swap(download)
	if lastUpload == upload && lastDownload == download {
		return Event{}, false
	}

	event := a.base
	event.Upload = upload
	event.Download = download
	return event, true
}
