// Package legacy uses two other flag vendors.
package legacy

import (
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	ld "github.com/launchdarkly/go-server-sdk/v7"
)

// DarkMode asks LaunchDarkly.
func DarkMode(c *ld.LDClient) bool {
	on, _ := c.BoolVariation("dark-mode", ldcontext.New("user"), false)
	return on
}
