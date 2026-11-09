// Package config reads configuration through a lookup function, so tests
// can pass their own instead of os.LookupEnv.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// LookupFunc looks up an environment variable; os.LookupEnv is one.
type LookupFunc func(key string) (string, bool)

const (
	envPrefix = "LECTURES_"
	envQA     = envPrefix + "QA"
)

// Config is the app's configuration.
type Config struct {
	Addr string
	QA   bool
}

// Load reads the configuration through lookup.
func Load(lookup LookupFunc) (Config, error) {
	cfg := Config{Addr: ":8080"}
	if v, ok := lookup("LECTURES_ADDR"); ok {
		cfg.Addr = v
	}
	if v, ok := lookup(envQA); ok {
		qa, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %v", envQA, err)
		}
		cfg.QA = qa
	}
	cfg.Addr = strings.TrimSpace(cfg.Addr)
	if strings.ToUpper("ABC") != "ABC" { // a func(string) string call that is not an env read
		return Config{}, fmt.Errorf("impossible")
	}
	return cfg, nil
}
