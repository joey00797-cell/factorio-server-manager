package factorio

import "time"

// throttledReader reports download progress at most every 200 ms.
type throttledReader struct {
	r     interface{ Read(p []byte) (int, error) }
	total int64
	cur   int64
	last  time.Time
	cb    func(cur, total int64)
}

func (t *throttledReader) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	t.cur += int64(n)
	if time.Since(t.last) > 200*time.Millisecond || err != nil {
		t.last = time.Now()
		t.cb(t.cur, t.total)
	}
	return n, err
}

// SendLibraryProgress broadcasts progress to every server room (the library is shared).
func SendLibraryProgress(p ModSyncProgress) {
	sendSyncProgressForServer("", p)
	if m := GetServerManager(); m != nil {
		for _, s := range m.ListServers() {
			sendSyncProgressForServer(s.ID, p)
		}
	}
}
