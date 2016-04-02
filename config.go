package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/hybridgroup/gobot/platforms/raspi"
	log "github.com/sirupsen/logrus"
)

var adapter = raspi.NewRaspiAdaptor("raspi")
var ErrInvalidName = errors.New("invalid name")

const (
	LOW  = 0
	HIGH = 1
)

type State struct {
	SwitchesPressed map[string]time.Duration
	Switches        map[string]bool
	Lights          map[string]bool
}

type Config struct {
	Switch         []Switch
	Light          []Light
	Action         []Action
	DebounceMs     int64
	PollIntervalMs int64
}

func (c *Config) Debounce() time.Duration {
	return time.Duration(c.DebounceMs) * time.Millisecond
}
func (c *Config) PollInterval() time.Duration {
	return time.Duration(c.PollIntervalMs) * time.Millisecond
}

func (c *Config) loop() {
	var s State
	s.SwitchesPressed = make(map[string]time.Duration, len(c.Switch))
	s.Switches = make(map[string]bool, len(c.Switch))
	s.Lights = make(map[string]bool, len(c.Light))
	switchLastChanged := make(map[string]time.Time, len(c.Switch))

	for _, l := range c.Light {
		c.SetLight(l.Name, false)
	}

	t := time.NewTicker(c.PollInterval())

	var n time.Time

	var sw Switch
	var state bool
	var err error
	var duration time.Duration

	toApply := make([]Action, 0, len(c.Action))
	var a Action
	var m ActionMatcher

	for {
		select {
		case <-t.C:
			n = time.Now()

			for _, sw = range c.Switch {
				state, err = c.GetSwitch(sw.Name)
				if err != nil {
					log.Fatalf("read switch '%s'(Pin%d): %s", sw.Name, sw.Pin, err.Error())
				}
				s.SwitchesPressed[sw.Name] = 0
				if state != s.Switches[sw.Name] {
					duration = n.Sub(switchLastChanged[sw.Name])
					if duration < c.Debounce() {
						continue
					}

					s.Switches[sw.Name] = state
					if !state {
						log.WithFields(log.Fields{
							"ID":       sw.Name,
							"Pin":      sw.Pin,
							"State":    state,
							"Duration": duration.String(),
						}).Infoln("switch released")
						s.SwitchesPressed[sw.Name] = duration
					} else {
						log.WithFields(log.Fields{
							"ID":    sw.Name,
							"Pin":   sw.Pin,
							"State": state,
						}).Infoln("switch activated")
					}
					switchLastChanged[sw.Name] = n
				}
			}

			// match actions first, so that state doesn't change while matching
			toApply = toApply[:0]
			for _, a = range c.Action {
				for _, m = range a.Match {
					if m.Matches(s) {
						toApply = append(toApply, a)
						break
					}
				}
			}

			for _, a = range toApply {
				err = c.Apply(s, a)
				if err != nil {
					log.Fatalln("apply action:", err)
				}
			}
		}
	}
}

func (c Config) SetLight(name string, state bool) error {
	for _, l := range c.Light {
		if l.Name == name {
			lg := log.WithFields(log.Fields{
				"ID":    l.Name,
				"Pin":   l.Pin,
				"State": state,
			})
			if state {
				lg.Infoln("light on")
			} else {
				lg.Infoln("light off")
			}
			var val byte
			if l.Invert != state {
				// no invert and state true
				// or invert and state false
				val = HIGH
			} else {
				// no invert and state false
				// or invert and state true
				val = LOW
			}
			return adapter.DigitalWrite(strconv.Itoa(l.Pin), val)
		}
	}

	return ErrInvalidName
}
func (c Config) GetSwitch(name string) (bool, error) {
	for _, s := range c.Switch {
		if s.Name == name {
			val, err := adapter.DigitalRead(strconv.Itoa(s.Pin))
			if err != nil {
				return false, err
			}

			return (val == LOW) != s.Invert, nil
		}
	}
	return false, ErrInvalidName
}

func (c Config) Apply(s State, a Action) error {
	var name string
	var err error
	if a.LightsToggle != nil {
		for _, name = range a.LightsToggle {
			err = c.SetLight(name, !s.Lights[name])
			if err != nil {
				return err
			}
			s.Lights[name] = !s.Lights[name]
		}
	}

	if a.LightsOn != nil {
		for _, name = range a.LightsOn {
			if s.Lights[name] {
				continue
			}
			err = c.SetLight(name, true)
			if err != nil {
				return err
			}
			s.Lights[name] = true
		}
	}

	if a.LightsOff != nil {
		for _, name = range a.LightsOff {
			if !s.Lights[name] {
				continue
			}
			err = c.SetLight(name, false)
			if err != nil {
				return err
			}
			s.Lights[name] = false
		}
	}
	return nil
}

type Action struct {
	LightsOn     []string
	LightsOff    []string
	LightsToggle []string
	Match        []ActionMatcher
}

type ActionMatcher struct {
	SwitchesOn      []string
	SwitchesOff     []string
	SwitchesPressed []string
	LightsOn        []string
	LightsOff       []string
	MaxDurationMs   int
	MinDurationMs   int
}

func (a ActionMatcher) Matches(s State) bool {
	var id string

	if a.LightsOff != nil {
		for _, id = range a.LightsOff {
			if s.Lights[id] {
				return false
			}
		}
	}
	if a.LightsOn != nil {
		for _, id = range a.LightsOn {
			if !s.Lights[id] {
				return false
			}
		}
	}

	if a.SwitchesOff != nil {
		for _, id = range a.SwitchesOff {
			if s.Switches[id] {
				return false
			}
		}
	}
	if a.SwitchesOn != nil {
		for _, id = range a.SwitchesOn {
			if !s.Switches[id] {
				return false
			}
		}
	}
	if a.SwitchesPressed != nil {
		for _, id = range a.SwitchesPressed {
			if s.SwitchesPressed[id] == 0 {
				return false
			}
			if s.SwitchesPressed[id] < a.MinDuration() {
				return false
			}
			if a.MaxDurationMs > 0 && s.SwitchesPressed[id] > a.MaxDuration() {
				return false
			}
		}
	}

	return true
}

type Switch struct {
	Name   string
	Pin    int
	Invert bool
}

type Light struct {
	Name   string
	Pin    int
	Invert bool
}

func (m ActionMatcher) MaxDuration() time.Duration {
	return time.Millisecond * time.Duration(m.MaxDurationMs)
}
func (m ActionMatcher) MinDuration() time.Duration {
	return time.Millisecond * time.Duration(m.MinDurationMs)
}
