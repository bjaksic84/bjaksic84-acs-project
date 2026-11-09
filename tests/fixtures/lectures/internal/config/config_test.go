package config

import "testing"

func TestLoad(t *testing.T) {
	env := map[string]string{"LECTURES_ADDR": ":9000", "LECTURES_QA": "true"}
	cfg, err := Load(func(k string) (string, bool) { v, ok := env[k]; return v, ok })
	if err != nil || cfg.Addr != ":9000" || !cfg.QA {
		t.Fatalf("Load = %+v, %v", cfg, err)
	}
}
