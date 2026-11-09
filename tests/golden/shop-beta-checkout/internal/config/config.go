// Package config reads the shop's configuration from the environment.
package config

import (
	"context"
	"os"
	"strconv"

	"example.com/shop/internal/flags"
)

// Config is the shop's configuration.
type Config struct {
	Addr            string
	Recommendations bool
	BetaCheckout    bool
}

// Load reads the configuration from the environment.
func Load() Config {
	cfg := Config{Addr: os.Getenv("SHOP_ADDR")}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	cfg.Recommendations = os.Getenv("FEATURE_RECOMMENDATIONS") == "true"
	beta, err := strconv.ParseBool(os.Getenv("SHOP_BETA_CHECKOUT"))
	if err != nil {
		beta = false
	}
	beta = flags.Enabled(context.Background(), flags.BetaCheckout, beta)
	cfg.BetaCheckout = beta
	return cfg
}
