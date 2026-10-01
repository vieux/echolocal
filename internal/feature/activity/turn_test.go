package activity

import (
	"testing"
	"time"

	"github.com/ygelfand/echolocal/internal/component"
)

// caught collects the events a turn fires, so a test can read what Home Assistant would have.
func caught(t *testing.T) *[]component.Event {
	t.Helper()

	var got []component.Event
	cancel := component.Fire.Listen(func(e component.Event) { got = append(got, e) })
	t.Cleanup(cancel)

	return &got
}

func TestDetectionOnlyReportsActivityWithoutVoicePhases(t *testing.T) {
	got := caught(t)
	Get().DetectedOnly("alfred_good_night", "Alfred Good Night", 1)
	Get().DetectedOnly("alfred_good_night", "Alfred Good Night", 1)
	if len(*got) != 4 {
		t.Fatalf("got %d events, want activity and wake event for each detection", len(*got))
	}
	ids := map[string]bool{}
	for _, i := range []int{0, 2} {
		e := (*got)[i]
		if e.Name != TurnEvent || e.Data["outcome"] != "completed" || e.Data["wake_word"] != "Alfred Good Night" || e.Data["slot"] != "1" || e.Data["version"] != Version {
			t.Fatalf("unexpected activity: %+v", e)
		}
		if e.Data["id"] == "" || ids[e.Data["id"]] {
			t.Fatal("each detection needs a unique activity id")
		}
		ids[e.Data["id"]] = true
		for _, key := range []string{"listen_ms", "think_ms", "speak_ms", "audio_seconds", "heard", "reply"} {
			if _, exists := e.Data[key]; exists {
				t.Errorf("detection-only activity contains %s", key)
			}
		}
		if (*got)[i+1].Name != "esphome.echolocal_wake_word" {
			t.Fatal("automation event missing")
		}
	}
}

// A phase is only reported if the turn reached it, which is the difference between "spent no time
// speaking" and "never got as far as speaking".
func TestAFailedTurnReportsNoPhaseItNeverReached(t *testing.T) {
	got := caught(t)

	turn := Get().Begin(1, "hey jarvis")
	time.Sleep(2 * time.Millisecond)
	turn.Listening()
	time.Sleep(2 * time.Millisecond)
	turn.Heard("what time is it")
	time.Sleep(2 * time.Millisecond)
	turn.Ends(Failed)

	if len(*got) != 1 {
		t.Fatalf("fired %d events, want 1", len(*got))
	}

	data := (*got)[0].Data
	for _, key := range []string{"listen_ms", "think_ms"} {
		if data[key] == "" {
			t.Errorf("%s missing, and the turn reached that phase", key)
		}
	}
	if data["speak_ms"] != "" {
		t.Errorf("speak_ms is %q, and the turn never replied", data["speak_ms"])
	}
	if data["reply"] != "" {
		t.Errorf("reply is %q, and there was none", data["reply"])
	}
	if data["outcome"] != string(Failed) {
		t.Errorf("outcome is %q, want %q", data["outcome"], Failed)
	}
	if data["heard"] != "what time is it" {
		t.Errorf("heard is %q", data["heard"])
	}
}

func TestAWholeTurnReportsEveryPhase(t *testing.T) {
	got := caught(t)

	turn := Get().Begin(2, "alexa")
	for _, step := range []func(){
		turn.Listening,
		func() { turn.Heard("play something") },
		func() { turn.Replying("playing") },
	} {
		time.Sleep(2 * time.Millisecond)
		step()
	}
	time.Sleep(2 * time.Millisecond)
	turn.Ends(Completed)

	data := (*got)[0].Data
	for _, key := range []string{"listen_ms", "think_ms", "speak_ms"} {
		if data[key] == "" {
			t.Errorf("%s missing", key)
		}
	}
	if data["slot"] != "2" {
		t.Errorf("slot is %q, want 2", data["slot"])
	}
	if data["id"] == "" {
		t.Error("no id, so nothing could ask for the recording")
	}
	if data["version"] != Version {
		t.Errorf("version is %q, want %q", data["version"], Version)
	}
}

// Several things notice a turn is over — the run closing, the reply finishing, a timeout — and the
// event has to be the one, not one each.
func TestEndingTwiceFiresOnce(t *testing.T) {
	got := caught(t)

	turn := Get().Begin(1, "hey jarvis")
	turn.Ends(Completed)
	turn.Ends(Timeout)

	if len(*got) != 1 {
		t.Fatalf("fired %d events, want 1", len(*got))
	}
	if (*got)[0].Data["outcome"] != string(Completed) {
		t.Errorf("outcome is %q, want the first one", (*got)[0].Data["outcome"])
	}
}

// A turn nobody started is what a cancel arriving while idle has, so every method takes a nil.
func TestNoTurnIsSafeToMark(t *testing.T) {
	got := caught(t)

	var turn *Turn
	turn.Listening()
	turn.Heard("nothing")
	turn.Replying("nothing")
	turn.Ends(Cancelled)

	if turn.ID() != "" {
		t.Errorf("id is %q, want empty", turn.ID())
	}
	if len(*got) != 0 {
		t.Errorf("fired %d events, want none", len(*got))
	}
}
