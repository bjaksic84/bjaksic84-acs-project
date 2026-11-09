// Package checkout already uses OpenFeature. Its dependencies are not in
// go.mod, so it does not type-check: scan must find the flags from syntax.
package checkout

import (
	"context"

	"github.com/open-feature/go-sdk/openfeature"
)

var client = openfeature.NewDefaultClient()

const bannerKey = "banner-text"

// NewCheckout reports whether the new checkout is on.
func NewCheckout(ctx context.Context) bool {
	on, _ := client.BooleanValue(ctx, "new-checkout", false, openfeature.EvaluationContext{})
	return on
}

// Banner returns the banner text.
func Banner(ctx context.Context) string {
	text, _ := client.StringValue(ctx, bannerKey, "", openfeature.EvaluationContext{})
	d, err := client.BooleanValueDetails(ctx, "new-checkout", false, openfeature.EvaluationContext{})
	if err == nil && d.Value {
		text += " (new checkout)"
	}
	return text
}
