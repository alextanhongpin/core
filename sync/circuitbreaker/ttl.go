package circuitbreaker

import "time"

type TTL struct {
	Value     int
	ExpiresAt time.Time
}

func (t *TTL) Add(n int) int {
	t.Load()
	t.Value += n
	return t.Value
}

func (t *TTL) SetExpiry(ttl time.Duration) {
	t.ExpiresAt = time.Now().Add(ttl)
}

func (t *TTL) Reset() {
	t.Value = 0
	t.ExpiresAt = time.Time{}
}

func (t *TTL) Load() int {
	if !t.ExpiresAt.IsZero() && !time.Now().Before(t.ExpiresAt) {
		t.Value = 0
	}

	return t.Value
}
