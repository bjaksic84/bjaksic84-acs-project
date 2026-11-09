package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("SHOP_ADDR", "")
	t.Setenv("FEATURE_RECOMMENDATIONS", "")
	t.Setenv("SHOP_BETA_CHECKOUT", "")
	cfg := Load()
	if cfg.Addr != ":8080" || cfg.Recommendations || cfg.BetaCheckout {
		t.Fatalf("Load() = %+v", cfg)
	}
}

func TestLoadToggles(t *testing.T) {
	t.Setenv("FEATURE_RECOMMENDATIONS", "true")
	t.Setenv("SHOP_BETA_CHECKOUT", "1")
	cfg := Load()
	if !cfg.Recommendations || !cfg.BetaCheckout {
		t.Fatalf("Load() = %+v", cfg)
	}
}
