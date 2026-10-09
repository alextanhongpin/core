package channel_test

import (
	"github.com/alextanhongpin/core/dsync/channel"
	"github.com/alextanhongpin/dbtx/testing/redistest"
	"os"
	"testing"
)

func TestMain(m *testing.M) { stop := redistest.Init(); code := m.Run(); stop(); os.Exit(code) }
func TestReplayAndRepeatedPayloads(t *testing.T) {
	client := redistest.Client(t)
	ch, err := channel.New(client)
	if err != nil {
		t.Fatal(err)
	}
	key := t.Name()
	for range 2 {
		if err := ch.Send(t.Context(), key, []byte("same")); err != nil {
			t.Fatal(err)
		}
	}
	first, err := ch.RecvAfter(t.Context(), key, "0-0", -1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ch.RecvAfter(t.Context(), key, first.ID, -1)
	if err != nil || first.ID == second.ID || string(second.Value) != "same" {
		t.Fatalf("%+v %v", second, err)
	}
}
func TestMalformedEntryReturnsError(t *testing.T) {
	client := redistest.Client(t)
	client.Do(t.Context(), "XADD", t.Name(), "*", "other", "value")
	ch := channel.MustNew(client)
	if _, err := ch.RecvAfter(t.Context(), t.Name(), "0-0", -1); err == nil {
		t.Fatal("invalid stream entry accepted")
	}
	if _, err := channel.New(nil); err == nil {
		t.Fatal("nil client accepted")
	}
}
