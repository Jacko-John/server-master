package service

import (
	"server-master/internal/config"
	"server-master/pkg/utils"
)

// DynamicPortRuntime holds the runtime state for one dynamic port service.
type DynamicPortRuntime struct {
	Config *config.DynamicPortServiceConfig
	Queue  *utils.Queue[string]
	Port   *PortService
}

// Container holds all business services of the application.
type Container struct {
	Subscription        *SubscriptionService
	File                *FileService
	PortServices        []*PortService
	DynamicPortRegistry map[string]*DynamicPortRuntime
	Ruleset             *RulesetService
}

// NewContainer initializes and returns all business services.
func NewContainer(cfg *config.Config, registry map[string]*DynamicPortRuntime, portServices []*PortService) *Container {
	return &Container{
		Subscription:        NewSubscriptionService(cfg, registry),
		File:                NewFileService(cfg),
		PortServices:        portServices,
		DynamicPortRegistry: registry,
		Ruleset:             NewRulesetService(cfg),
	}
}
