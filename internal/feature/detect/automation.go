package detect

import (
	"fmt"
	"log/slog"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/echolocal/internal/config"
	"github.com/ygelfand/echolocal/internal/feature/wakeword"
	"github.com/ygelfand/echolocal/internal/lib/wake"
)

const automationOff = "None"

// Models already on the device can be used without assigning an assistant pipeline.
func automationOptions(models []wake.Model) []string {
	options := []string{automationOff}
	for _, m := range models {
		options = append(options, m.ID)
	}
	return options
}

func (d *Detect) loadAutomation(slot int, id string) error {
	if !wakeword.IsAutomationSlot(slot) {
		return fmt.Errorf("invalid automation slot %d", slot)
	}
	if id == "" {
		d.engine.Clear(slot)
		return nil
	}
	m, ok := wake.Find(wake.Lib().Ours(), id)
	if !ok {
		return fmt.Errorf("model %q is not installed", id)
	}
	return d.engine.Use(slot, m)
}

func (d *Detect) newAutomationEntities() {
	for n := range wakeword.AutomationSlots {
		slot := wakeword.Slots + n
		base := func(suffix, label string) esphome.Base {
			return esphome.Base{ObjectID: fmt.Sprintf("automation_phrase_%d_%s", n+1, suffix),
				Name: fmt.Sprintf("Automation phrase %d %s", n+1, label), Category: esphome.CategoryConfig}
		}
		model := &esphome.Select{Base: base("model", "model"), Options: automationOptions(wake.Lib().Ours())}
		threshold := &esphome.Number{Base: base("threshold", "sensitivity"), Min: 0.5, Max: 0.99, Step: 0.01, Mode: esphome.NumberBox}
		model.OnCommand = func(value string) {
			d.automationMu.Lock()
			defer d.automationMu.Unlock()
			id := value
			if id == automationOff {
				id = ""
			}
			for other := wakeword.Slots; other < wakeword.Slots+wakeword.AutomationSlots; other++ {
				if other != slot && id != "" && config.Get().Wake.Slot(other).ID == id {
					slog.Warn("automation phrase already selected", "id", id)
					return
				}
			}
			previous := config.Get().Wake.Slot(slot).ID
			if err := d.loadAutomation(slot, id); err != nil {
				slog.Error("selecting automation phrase", "err", err)
				return
			}
			if err := config.Set().Wake(slot).ID(id); err != nil {
				_ = d.loadAutomation(slot, previous)
				slog.Error("saving automation phrase", "err", err)
				return
			}
			model.Set(value)
		}
		threshold.OnCommand = func(value float32) {
			if err := config.Set().Wake(slot).Threshold(float64(value)); err != nil {
				slog.Error("saving automation sensitivity", "err", err)
				return
			}
			threshold.Set(value)
		}
		d.automations = append(d.automations, model, threshold)
	}
}

func (d *Detect) restoreAutomations(c config.Config) {
	for n := range wakeword.AutomationSlots {
		word := c.Wake.Slot(wakeword.Slots + n)
		value := word.ID
		if value == "" {
			value = automationOff
		}
		d.automations[n*2].(*esphome.Select).Set(value)
		d.automations[n*2+1].(*esphome.Number).Set(float32(word.Threshold))
	}
}
