package voice

import (
	"strconv"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/component"
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/activity"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
	"github.com/ygelfand/echolocal/internal/lib/wake"
)

func TestDisabledAssistantsStayDisabledWithAutomationSlots(t *testing.T) {
	saved := config.Wake{Words: []config.WakeWord{
		config.DefaultWakeWord(), config.DefaultWakeWord(), config.DefaultWakeWord(),
	}}
	saved.Words[2].ID = "alfred_good_night"
	models := []wake.Model{{ID: wake.DefaultModel}, {ID: "alfred_good_night"}}
	if got := selectedWakeWords(models, wakeword.Slots, saved); len(got) != 0 {
		t.Fatalf("disabled assistants restored as %v", got)
	}
	c := &conversation{vs: &esphome.VoiceSatellite{}}
	// Button requests must return before touching microphone, sound, or pipeline hardware.
	c.handle(event{kind: evStart, slot: 0})
	c.handle(event{kind: evStart, slot: 1})
	if c.turn != nil || c.phase != phaseIdle {
		t.Fatal("disabled assistant started a conversation")
	}
}

func TestExtraAutomationSlotsNeverStartConversation(t *testing.T) {
	for slot := wakeword.Slots; slot < wakeword.Slots+wakeword.AutomationSlots; slot++ {
		for _, p := range []phase{phaseIdle, phaseListening, phaseThinking, phaseReplying} {
			ack := -1
			c := &conversation{log: activity.Get(), phase: p, events: make(chan event, 1), acknowledge: func(n int) { ack = n }}
			var got []component.Event
			cancel := component.Fire.Listen(func(e component.Event) { got = append(got, e) })
			v := &Voice{turn: c}
			v.AutomationDetected(slot, "good_morning", "Good Morning")
			c.handle(<-c.events)
			cancel()
			if ack != wakeword.AutomationFeedbackSlot || c.phase != p || c.turn != nil || c.deadline != nil || c.pending != nil {
				t.Fatal("automation changed the conversation")
			}
			if len(got) != 2 || got[0].Name != activity.TurnEvent || got[1].Data["model"] != "good_morning" || got[1].Data["slot"] != strconv.Itoa(slot+1) {
				t.Fatalf("bad events: %+v", got)
			}
		}
	}
}

func TestAlfredDetectionReportsWithoutStartingOrInterruptingTurn(t *testing.T) {
	for _, p := range []phase{phaseIdle, phaseListening, phaseThinking, phaseReplying} {
		t.Run(p.String(), func(t *testing.T) {
			log := activity.Get()
			log.Woke("Okay Nabu")
			acknowledged := 0
			c := &conversation{
				vs:  &esphome.VoiceSatellite{ActiveWakeWords: []string{"okay_nabu", "alfred_good_night"}},
				log: log, phase: p,
				acknowledge: func(slot int) {
					if slot != 1 {
						t.Errorf("acknowledged slot %d, want 1", slot)
					}
					acknowledged++
				},
			}
			var events []component.Event
			cancel := component.Fire.Listen(func(e component.Event) {
				if e.Name == "esphome.echolocal_wake_word" {
					events = append(events, e)
				}
			})
			defer cancel()
			// No hardware or pipeline is attached: touching the normal turn path would fail.
			c.handle(event{kind: evDetected, slot: 1})
			c.handle(event{kind: evDetected, slot: 1})
			if acknowledged != 2 {
				t.Fatalf("got %d acknowledgements, want 2", acknowledged)
			}
			if c.phase != p || c.turn != nil || c.pending != nil || c.deadline != nil {
				t.Fatal("automation phrase changed conversation state")
			}
			if got := log.Entities()[0].(*esphome.TextSensor).Get(); got != "Alfred Good Night" {
				t.Fatalf("last wake word = %q", got)
			}
			if len(events) != 2 {
				t.Fatalf("got %d events for two detections", len(events))
			}
			for _, e := range events {
				if e.Name != "esphome.echolocal_wake_word" || e.Data["model"] != "alfred_good_night" || e.Data["wake_word"] != "Alfred Good Night" || e.Data["slot"] != "2" {
					t.Fatalf("unexpected event: %+v", e)
				}
			}
		})
	}
}

func TestDetectionAndManualWakeUseSeparateEvents(t *testing.T) {
	c := &conversation{events: make(chan event, 2)}
	v := &Voice{turn: c}
	v.Detected(1)
	v.Start(1)
	for _, want := range []eventKind{evDetected, evStart} {
		if got := <-c.events; got.kind != want || got.slot != 1 {
			t.Fatalf("got %+v, want event %v for slot 1", got, want)
		}
	}
}

// What a device advertises as active when nothing has been chosen decides whether a fresh install can
// be spoken to at all, and it must not come down to which model sorts first.
func TestWakeWordsPreselectsTheDefault(t *testing.T) {
	for name, tc := range map[string]struct {
		installed []string
		want      string
	}{
		"the default is installed":     {[]string{"hey_jarvis", wake.DefaultModel, "hey_mycroft"}, wake.DefaultModel},
		"the default sorts last":       {[]string{"alexa", wake.DefaultModel}, wake.DefaultModel},
		"the default is not installed": {[]string{"hey_jarvis"}, "hey_jarvis"},
		"nothing installed":            {nil, ""},
	} {
		models := make([]wake.Model, 0, len(tc.installed))
		for _, id := range tc.installed {
			models = append(models, wake.Model{ID: id, Phrase: id})
		}

		active := activeWakeWords(models, wakeword.Slots)

		switch {
		case tc.want == "":
			if len(active) != 0 {
				t.Errorf("%s: listening for %v with nothing installed", name, active)
			}
		case len(active) != 1 || active[0] != tc.want:
			t.Errorf("%s: listening for %v, want just %q", name, active, tc.want)
		}
	}
}
