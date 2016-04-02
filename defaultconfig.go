package main

const DefaultDebounceMs = 25

const configFile = `
# NOTE: Pins are in reference to physical pin numbers

# Debounce will ignore subsequent button state changes for a duration
DebounceMs = 25

[[Switch]]
	Name = "SW1"
	Pin = 3
[[Switch]]
	Name = "SW2"
	Pin = 5
[[Switch]]
	Name = "SW3"
	Pin = 7
	# Uncomment to interpret pin `HIGH` state as switch being activated instead of `LOW` (the default)
	# Invert = true

[[Light]]
	Name = "LED1"
	Pin = 8
	# Uncomment to output signal `LOW` when active instead of `HIGH`
	# Invert = true
[[Light]]
	Name = "LED2"
	Pin = 10
[[Light]]
	Name = "LED3"
	Pin = 12

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
