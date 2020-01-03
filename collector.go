package main

import (
	"TL-Data-Collector/config"
	"flag"

	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
)

var (
	action = flag.String("action", "", "Service action for collector: start, stop, restart, install, uninstall")
)

func main() {
	flag.Parse()

	// init the settings
	var settings config.Config
	// parse the config file
	if err := config.ParseYamlFile("config.yml", &settings); err != nil {
		panic(err)
	}

	// define the service config for program
	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	serviceConfig := &service.Config{
		Name:         "TraderLinkedDataCollector",
		DisplayName:  "Trader Linked Data Collector Service",
		Description:  "The service that collects information from trading platforms",
		Dependencies: []string{},
		Option:       options,
	}

	prog := NewProgram(&settings)
	// new the service for program
	s, err := service.New(prog, serviceConfig)
	if err != nil {
		panic(err)
	}

	// if it's the service action
	if len(*action) > 0 {
		if err := service.Control(s, *action); err != nil {
			panic(err)
		}
		return
	}

	// start the service
	if err := s.Run(); err != nil {
		log.Error(err)
	}
}
