package pusher

import (
	"errors"
	"fmt"
	"sync"
)

// subscribedChannels contains subscribed channels
type subscribedChannels struct {
	channels []string
	sync.Mutex
}

// contains checks if channel is subscribed
func (s *subscribedChannels) contains(channel string) bool {
	for _, c := range s.channels {
		if c == channel {
			return true
		}
	}
	return false
}

// add a new cahnnel to the slice
func (s *subscribedChannels) add(channel string) error {
	s.Lock()
	defer s.Unlock()
	if s.contains(channel) {
		return errors.New(fmt.Sprintf("Channel %s already subscribed", channel))
	}
	s.channels = append(s.channels, channel)
	return nil
}

// remove a channel (if present) to the slice
func (s *subscribedChannels) remove(channel string) {
	s.Lock()
	defer s.Unlock()
	for i, c := range s.channels {
		if c == channel {
			s.channels = append(s.channels[:i], s.channels[i+1:]...)
		}
	}
}
