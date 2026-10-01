package detect

import (
	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
	"github.com/ygelfand/echolocal/internal/lib/wake"
	esphome "github.com/ygelfand/go-esphome-device"
	"testing"
)

func TestAutomationEntitiesRestoreSeparateSlots(t *testing.T) {
	d := &Detect{}
	d.newAutomationEntities()
	if len(d.automations) != 2*wakeword.AutomationSlots {
		t.Fatal("missing automation controls")
	}
	c := config.Config{}
	for range wakeword.Slots + wakeword.AutomationSlots {
		c.Wake.Words = append(c.Wake.Words, config.DefaultWakeWord())
	}
	c.Wake.Words[0].ID = "okay_nabu"
	c.Wake.Words[2].ID = "alfred_good_night"
	c.Wake.Words[2].Threshold = 0.9
	d.restoreAutomations(c)
	if got := d.automations[0].(*esphome.Select).Get(); got != "alfred_good_night" {
		t.Fatalf("model = %q", got)
	}
	if got := d.automations[1].(*esphome.Number).Get(); got != float32(0.9) {
		t.Fatalf("threshold = %v", got)
	}
	if got := d.automations[2].(*esphome.Select).Get(); got != automationOff {
		t.Fatalf("unused slot = %q", got)
	}
}

func TestDisablingAutomationLeavesAssistantAndStopLoaded(t *testing.T) {
	fakes(t)
	e := New(StopSlot+1, nil)
	for _, n := range []int{0, wakeword.Slots, StopSlot} {
		if err := e.Use(n, model("shared", wake.KindMicroWakeWord)); err != nil {
			t.Fatal(err)
		}
	}
	d := &Detect{engine: e}
	if err := d.loadAutomation(wakeword.Slots, ""); err != nil {
		t.Fatal(err)
	}
	if e.slots[wakeword.Slots].loaded || !e.slots[0].loaded || !e.slots[StopSlot].loaded {
		t.Fatal("wrong slots cleared")
	}
	if err := d.loadAutomation(StopSlot, ""); err == nil {
		t.Fatal("accepted reserved slot")
	}
}
