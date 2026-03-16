//go:build !enterprise

package main

import "go.uber.org/fx"

// enterpriseModule is an empty module when built without the enterprise tag.
var enterpriseModule = fx.Module("enterprise")
