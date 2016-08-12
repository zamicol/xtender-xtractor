package main

import xx "xtender-xtractor/lib"

var (
	configFile = "config.json" //Configuration file.
)

//main opens and parses the config, Starts logging, and then call setup
func main() {
	//Load Config
	c := new(xx.Configuration)
	c.Parse(configFile)

	//Process file
	c.Process()
}
