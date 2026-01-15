package agent

import (
	"fmt"
	"sync"
)

// AgentFactory is a function that creates an agent given a context
type AgentFactory func(ctx *Context) Agent

var (
	registry     = make(map[string]AgentFactory)
	registryLock sync.RWMutex
)

// Register adds an agent factory to the registry
func Register(agentType string, factory AgentFactory) {
	registryLock.Lock()
	defer registryLock.Unlock()
	registry[agentType] = factory
}

// Get retrieves an agent factory from the registry
func Get(agentType string) (AgentFactory, error) {
	registryLock.RLock()
	defer registryLock.RUnlock()
	factory, exists := registry[agentType]
	if !exists {
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
	return factory, nil
}

// Has checks if an agent type is registered
func Has(agentType string) bool {
	registryLock.RLock()
	defer registryLock.RUnlock()
	_, exists := registry[agentType]
	return exists
}

// List returns all registered agent types
func List() []string {
	registryLock.RLock()
	defer registryLock.RUnlock()
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
