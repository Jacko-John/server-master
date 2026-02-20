package service

import (
	"server-master/internal/config"
	"server-master/pkg/utils"
)

// Container holds all business services of the application.
type Container struct {
	Subscription *SubscriptionService
	File         *FileService
	Port         *PortService
	Ruleset      *RulesetService
}

// NewContainer initializes and returns all business services.
func NewContainer(cfg *config.Config, queue *utils.Queue[string]) *Container {
	return &Container{
		Subscription: NewSubscriptionService(cfg, queue),
		File:         NewFileService(cfg),
		Port:         NewPortService(cfg, queue),
		Ruleset:      NewRulesetService(cfg),
	}
}
