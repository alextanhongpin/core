package snapshot_test

import (
	"github.com/alextanhongpin/core/sync/snapshot"
	"testing"
	"time"
)

func TestConfigDefaultsAndValidation(t *testing.T) {
	s, stop, err := snapshot.New(snapshot.Config{})
	if err != nil || s == nil {
		t.Fatal(err)
	}
	stop()
	for _, cfg := range []snapshot.Config{{BufferSize: -1}, {Policies: []snapshot.Policy{}}, {Policies: []snapshot.Policy{{Changes: 0}}}, {Policies: []snapshot.Policy{{Changes: 1, After: -time.Second}}}} {
		s, stop, err := snapshot.New(cfg)
		if err == nil || s != nil || stop != nil {
			t.Fatalf("accepted invalid configuration: %+v", cfg)
		}
	}
}
