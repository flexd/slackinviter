package main

import (
	"log"
	"sync"

	"github.com/nlopes/slack"
)

// Team information
type team struct {
	mu      sync.RWMutex
	iconURL string
	name    string
	domain  string
}

// TODO(freeformz): default image?
func (t *team) Update(s *slack.TeamInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.name = s.Name
	t.domain = s.Domain
	if v, ok := s.Icon["image_default"]; ok {
		if b, ok := v.(bool); ok && b {
			t.iconURL = ""
			return
		}
	}
	var icons = []string{"132", "102", "88", "68", "44", "34"}
	for _, i := range icons {
		img, ok := s.Icon["image_"+i]
		if ok {
			if str, ok := img.(string); ok {
				t.iconURL = str
			}
			return
		}
	}
	log.Println("Unable to determine icon image")
}

//Icon information for the teams
func (t *team) Icon() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.iconURL
}

// Name of the team
func (t *team) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// Domain of the team
func (t *team) Domain() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.domain
}
