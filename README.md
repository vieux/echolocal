# EchoLocal

<a href="https://github.com/ygelfand/echolocal/stargazers"><img src="https://img.shields.io/github/stars/ygelfand/echolocal?style=for-the-badge&label=Stars&color=d6a102" alt="Stars"></a>
<a href="https://github.com/ygelfand/echolocal/releases"><img src="https://img.shields.io/github/downloads/ygelfand/echolocal/total?style=for-the-badge&label=Downloads&color=e8604c" alt="Downloads"></a>
<a href="https://github.com/ygelfand/echolocal/releases/latest"><img src="https://shields.io/github/v/release/ygelfand/echolocal?style=for-the-badge&color=5da3a6" alt="Version"></a>
<a href="https://github.com/ygelfand/echolocal/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/ygelfand/echolocal/ci.yml?style=for-the-badge&label=Build&color=3fbf5f" alt="Build"></a>
<a href="https://buymeacoffee.com/ygelfand"><img src="https://img.shields.io/badge/Buy_me_a_coffee-ffdd00?style=for-the-badge&logo=buymeacoffee&logoColor=000" alt="Buy me a coffee"></a>

Upgrade your 2nd-generation Amazon Echo Dot (biscuit, RS03QR) into a local Home Assistant voice satellite (and more).

A pure-Go replacement for Amazon's services that speaks the ESPHome native API, so Home Assistant
discovers it the way it discovers any ESPHome device.

Requires a device unlocked with TWRP or similar — see [xdaforums](https://xdaforums.com/t/unlock-root-twrp-unbrick-amazon-echo-dot-2nd-gen-2016-biscuit.4761416/) for details.

## Includes:

**100% on-device local wake words.** Supports [openWakeWord](https://github.com/dscripka/openWakeWord)
and [microWakeWord](https://github.com/kahrendt/microWakeWord) models, including "stop" detection.

**LED ring.** Twelve individually addressable segments, multiple animation effects across ambient, motion, alert and
room-reactive behavior, and a color picker per segment. The ring can follow the room's volume.

**Speaker.** A native Home Assistant `media_player` — play any media Home Assistant can hand it.
Music can duck under a wake word, and resumes. Also supports white-noise generation.

**Bluetooth proxy.** BLE advertisements forwarded to Home Assistant, so the Dot extends your
Bluetooth and can be used in integrations like [bermuda](https://github.com/agittins/bermuda).
While the proxy is enabled, the Dot also advertises as an iBeacon with UUID
`9c5fa6f1-91c4-4f56-bb9f-d92acfd9d40b`, major `1`, and a minor derived from the final two bytes of
its factory MAC address. The beacon stops when the proxy is disabled. This supports tools such as
[BLE Positioning System](https://github.com/Hogster/BPS), which use the physical positions of BLE
receivers to locate tracked devices.

**Lux sensor.** The board carries an ambient light sensor that Amazon appears to have left unused.

## The optional integration

Everything above works with stock Home Assistant. The
[EchoLocal integration](https://github.com/ygelfand/echolocal-hacs) (HACS) adds optional functionality:

- a dashboard and cards built for these devices
- per-turn history
- play back of what the microphones heard
- a wake word library manager

![The EchoLocal dashboard: three Dots, rings lit, each showing its room's light level](docs/images/echolocal_dash_lit.png)

## Installing

You need a 2nd-generation Echo Dot, connected via a USB cable, and a device that has been unlocked with TWRP as its
recovery partition. `echoctl` does the rest. It'll prompt for wifi configuration if it hasn't been setup and provide espHome encryption key

```sh
echoctl install --name living-room
```

![echoctl install, from flashing the boot image to the device coming back on wifi](docs/images/install.gif)

It then turns up in Home Assistant on its own, and the key `echoctl` printed is what pairs it:

<p align="center">
  <img src="docs/images/echolocal_discovery.png" alt="Home Assistant discovering the device as an ESPHome node" height="230">
  <img src="docs/images/echolocal_discovery_add.png" alt="The confirmation dialog for adding the discovered device" height="230">
</p>

## Automation phrases

There are three independent **Automation phrase** slots on the main ESPHome device's
configuration page. Each has a model selector (`None` disables it) and its own sensitivity.
All automation phrases share Assistant 2's wake tone and ring effect, configurable in the EchoLocal
dashboard even when Assistant 2's wake word is None. Ring color uses the shared ring setting.
Previously saved per-automation tones and effects are ignored. These slots use installed models
without consuming the two assistant pipeline slots. After
installing new model files, restart echod to refresh the selectors. Select each model in only one
automation slot; if it is also selected in an assistant slot, the automation takes precedence.
The custom EchoLocal assistant picker still shows only the two assistant slots.
For automation-only use, set both assistant wake words to None. This stays disabled after restart
and prevents the Wake/action buttons from starting conversations. Automation phrase slots still run.

Every automation phrase updates Last wake word, plays Assistant 2's configured feedback, records Activity,
and emits `esphome.echolocal_wake_word`. Match `model` to the selected model ID in HA automations.
Event `slot` values are `"3"`, `"4"`, and `"5"` for automation slots 1–3. Unused slots do not load
models. Enable models one at a time and check detection performance; each distinct model adds CPU work.

This build bundles the following models. Model IDs are case-sensitive and are also the values
used by the automation selectors and the event's `model` field.

| Model ID | Phrase |
| --- | --- |
| `hey_alfred` | Hey Alfred |
| `alfred_dark` | Alfred Dark |
| `alfred_lights` | Alfred Lights |
| `alfred_good_night` | Alfred Good Night |
| `alfred_lights_on` | Alfred Lights On |
| `alfred_lights_off` | Alfred Lights Off |
| `alfred_movie` | Alfred Movie |
| `alfred_movie_time` | Alfred Movie Time |
| `alfred_reading_time` | Alfred Reading Time |

The bundled `alfred_good_night` model is an automation phrase. Select **Alfred Good Night** in a
wake-word slot to listen for it. Detection updates **Last wake word** to `Alfred Good Night`
without opening a conversation or interrupting an existing turn. It plays Assistant 2's configured
wake tone and ring effect, with the ring returning to its previous state after 1.5 seconds.
Other assistant wake words and manual Wake buttons still start conversations when an assistant is enabled.
Each detection appears in Activity as a completed 0.0s entry with no recording or voice phases.

Each detection also emits `esphome.echolocal_wake_word` with `model: alfred_good_night`,
`wake_word: Alfred Good Night`, and `slot` (a one-based string). Use this event for automations
that must run on consecutive detections of the same phrase; the text sensor value may be unchanged.

## Building it yourself

`make dist` builds the device binaries and an `echoctl` containing the bundled models.
Use `bin/echoctl install --manifest bin/manifest.json` to install both on a connected device.
The installer adds missing models; it does not overwrite existing ones or remove old models.
`make install-echod` updates only the daemon.

```sh
make build-echod     # cross-compile the daemon for the Dot
make install-echod   # build, install, and restart it on a connected device
```

## How it fits together

- **echod** runs on the Dot: the hardware, the wake word engines, the conversation, and an ESPHome
  native API server.
- **echoctl** is the host CLI: installing, re-installing, provisioning, and offline tools for debugging the device.
- **[go-esphome-device](https://github.com/ygelfand/go-esphome-device)** implements the device half of
  the ESPHome protocol, including the voice satellite and Bluetooth proxy.
