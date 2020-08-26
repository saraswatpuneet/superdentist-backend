package options

import (
	"encoding/json"
	"fmt"
)
// Options .. contains global options like ones read from environment variables
type Options struct {
	Debug          bool   `json:"debug,omitempty"`
	MaxPayloadSize int64  `json:"max_payload_size,omitempty"`
	MaxHeaderSize  int    `json:"max_header_size,omitempty"`
}

// New .. create a new instance 
func New() *Options {
	return &Options{}
}

// InitOptions initializes the options
func InitOptions() (*Options, error) {
	options := New()
	if err := json.Unmarshal(Default, options); err != nil {
		return nil, fmt.Errorf("Options initialization unmarshal error: %v", err)
	}
	return options, nil
}
