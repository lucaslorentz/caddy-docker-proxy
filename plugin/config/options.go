package config

import "time"

// Options are the options for generator
type Options struct {
	CaddyFilePath     string
	LabelPrefix       string
	ProxyServiceTasks bool
	ValidateNetwork   bool
	ProcessCaddyfile  bool
	PollingInterval   time.Duration
}
