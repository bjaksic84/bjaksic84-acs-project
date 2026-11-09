package legacy

import "github.com/Unleash/unleash-go-sdk/v5"

// BetaSearch asks Unleash.
func BetaSearch() bool {
	return unleash.IsEnabled("beta-search")
}
