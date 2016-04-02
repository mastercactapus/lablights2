package main

import (
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	installPrefix string
	installReset  bool
	configPath    string

	mainCmd = &cobra.Command{}
	runCmd  = &cobra.Command{
		Use: "run",
		Run: runLights,
	}
	installCmd = &cobra.Command{
		Use: "install",
		Run: runInstall,
	}
)

func runInstall(cmd *cobra.Command, args []string) {
	err := install(installPrefix, installReset)
	if err != nil {
		log.Fatalln("install:", err)
	}
}

func missingID(ids map[string]bool, check []string) string {
	for _, id := range check {
		if !ids[id] {
			return id
		}
	}
	return ""
}

func checkMatch(lights, switches map[string]bool, actionIndex, index int, m ActionMatcher) {
	check := func(ids map[string]bool, items []string, itype, name string) {
		missing := missingID(ids, items)
		if missing != "" {
			log.Fatalf("unknown %s identifier '%s' in action #%d matcher #d.%s", itype, missing, actionIndex, index, name)
		}
	}

	check(lights, m.LightsOff, "light", "LightsOff")
	check(lights, m.LightsOn, "light", "LightsOn")
	check(switches, m.SwitchesOff, "switch", "SwitchesOff")
	check(switches, m.SwitchesOn, "switch", "SwitchesOn")
	check(switches, m.SwitchesPressed, "switch", "SwitchesPressed")
}

func (c Config) Validate() {

	if c.Action == nil {
		log.Fatalln("no actions configured, aborting")
	}
	if c.Light == nil {
		log.Fatalln("no lights configured, aborting")
	}
	if c.Switch == nil {
		log.Fatalln("no switches configured, aborting")
	}
	if c.DebounceMs == 0 {
		c.DebounceMs = DefaultDebounceMs
	}

	// check that IDs exist
	lights := make(map[string]bool, len(c.Light))
	switches := make(map[string]bool, len(c.Switch))
	for _, l := range c.Light {
		lights[l.Name] = true
	}
	for _, sw := range c.Switch {
		switches[sw.Name] = true
	}

	for i, a := range c.Action {
		if a.Match == nil || len(a.Match) == 0 {
			log.Warnf("no matchers for action #%d, it will never run", i)
			continue
		}
		if a.LightsOff != nil {
			missing := missingID(lights, a.LightsOff)
			if missing != "" {
				log.Fatalf("unknown light identifier '%s' in action #%d.LightsOff", missing, i)
			}
		}
		if a.LightsOn != nil {
			missing := missingID(lights, a.LightsOn)
			if missing != "" {
				log.Fatalf("unknown light identifier '%s' in action #%d.LightsOn", missing, i)
			}
		}
		if a.LightsToggle != nil {
			missing := missingID(lights, a.LightsToggle)
			if missing != "" {
				log.Fatalf("unknown light identifier '%s' in action #%d.LightsToggle", missing, i)
			}
		}

		for mi, m := range a.Match {
			checkMatch(lights, switches, i, mi, m)
		}
	}
}

func runLights(cmd *cobra.Command, args []string) {
	var c Config
	_, err := toml.DecodeFile(configPath, &c)

	if err != nil {
		log.Fatalln("load config:", err)
	}
	c.Validate()

	c.loop()
}

func main() {
	installCmd.Flags().BoolVar(&installReset, "reset", false, "Reset config. Resets configuration to default, even if a config file already exists")
	installCmd.Flags().StringVarP(&installPrefix, "prefix", "p", "", "Install prefix. Prefix to install directory, default is /")
	mainCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "/etc/lablights2.conf", "Config path. The path to the configuration file")
	mainCmd.AddCommand(runCmd, installCmd)
	mainCmd.Execute()
}
