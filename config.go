package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/hybridgroup/gobot/platforms/raspi"
)

const configFile = `
# NOTE: Pins are in reference to GPIO numbers

# Debounce will ignore subsequent button state changes for a duration
DebounceMs = 25

[[Switch]]
	Name = "SW1"
	Pin = 14
[[Switch]]
	Name = "SW2"
	Pin = 15
[[Switch]]
	Name = "SW3"
	Pin = 16
	# Uncomment to interpret pin "high" state as switch being activated instead of "low"
	# Invert = true

[[Light]]
	Name = "LED1"
	Pin = 17
	# Uncomment to output signal "high" when active instead of "low"
	# Invert = true
[[Light]]
	Name = "LED2"
	Pin = 18
[[Light]]
	Name = "LED3"
	Pin = 19

[[Action]]
	# toggle LED1 if SW1 was pressed for > 1s
	Toggle = ["LED1"]
	[[Action.Match]]
		SwitchesPressed = [ "SW1" ]
		MinDurationMs = 1000
[[Action]]
	# toggle LED2 if SW2 was pressed for > 1s
	LightsToggle = ["LED2"]
	[[Action.Match]]
		SwitchesPressed = [ "SW2" ]
		MinDurationMs = 1000
[[Action]]
	# turn off both LED1 and LED2 if either SW1 or SW2 was pressed < 1s and either LED1 or LED2 are on
	LightsOff = ["LED1", "LED2"]
	[[Action.Match]]
		SwitchesPressed = [ "SW1" ]
		MaxDurationMs = 1000
		LightsOn = ["LED1"]
	[[Action.Match]]
		SwitchesPressed = [ "SW1" ]
		MaxDurationMs = 1000
		LightsOn = ["LED2"]
	[[Action.Match]]
		SwitchesPressed = [ "SW2" ]
		MaxDurationMs = 1000
		LightsOn = ["LED1"]
	[[Action.Match]]
		SwitchesPressed = [ "SW2" ]
		MaxDurationMs = 1000
		LightsOn = ["LED2"]
[[Action]]
	# turn on both LED1 and LED2 if either SW1 or SW2 was pressed < 1s and both LED1 or LED2 are off
	LightsOn = ["LED1", "LED2"]
	[[Action.Match]]
		SwitchesPressed = [ "SW1" ]
		MaxDurationMs = 1000
		LightsOff = ["LED1", "LED2"]
	[[Action.Match]]
		SwitchesPressed = [ "SW2" ]
		MaxDurationMs = 1000
		LightsOff = ["LED1", "LED2"]

[[Action]]
	# turn LED3 on when SW3 is on
	LightsOn = ["LED3"]
	[[Action.Match]]
		SwitchesOn = ["SW3"]
[[Action]]
	# turn LED3 on when SW3 is off
	LightsOff = ["LED3"]
	[[Action.Match]]
		SwitchesOff = ["SW3"]
`

var adapter = raspi.NewRaspiAdaptor("raspi")
var ErrInvalidName = errors.New("invalid name")

const (
	LOW  byte = 0
	HIGH      = 1
)

type State struct {
	SwitchesPressed map[string]time.Duration
	Switches        map[string]bool
	Lights          map[string]bool
}

type Config struct {
	Switch     []Switch
	Light      []Light
	Action     []Action
	DebounceMs int64
}

func (c *Config) Debounce() time.Duration {
	return time.Duration(c.DebounceMs) * time.Millisecond
}

func (c *Config) loop() {
	var s State
	s.SwitchesPressed = make(map[string]time.Duration, len(c.Switch))
	s.Switches = make(map[string]bool, len(c.Switch))
	s.Lights = make(map[string]bool, len(c.Light))
	switchLastChanged := make(map[string]time.Time, len(c.Switch))

	t := time.NewTicker(time.Millisecond)

	for {
		select {
		case <-t.C:
			n := time.Now()

			for _, sw := range c.Switch {
				state, err := c.GetSwitch(sw.Name)
				if err != nil {
					panic(err)
				}
				s.SwitchesPressed[sw.Name] = 0
				if state != s.Switches[sw.Name] {
					duration := n.Sub(switchLastChanged[sw.Name])
					if duration < c.Debounce() {
						continue
					}

					s.Switches[sw.Name] = state
					if !state {
						s.SwitchesPressed[sw.Name] = duration
					}
					switchLastChanged[sw.Name] = n
				}
			}
		}
	}
}

func (c Config) SetLight(name string, state bool) error {
	for _, l := range c.Light {
		if l.Name == name {
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

			return (val == HIGH) != s.Invert, nil
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
		}
	}

	if a.LightsOn != nil {
		for _, name = range a.LightsOn {
			err = c.SetLight(name, true)
			if err != nil {
				return err
			}
		}
	}

	if a.LightsOff != nil {
		for _, name = range a.LightsOff {
			err = c.SetLight(name, false)
			if err != nil {
				return err
			}
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
			if s.SwitchesPressed[id] > a.MaxDuration() {
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
