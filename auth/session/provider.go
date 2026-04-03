package session

import "github.com/google/wire"

// ProviderSet is the Wire provider set for the session package.
var ProviderSet = wire.NewSet(
	NewManager,
	NewMemoryStore,
	wire.Bind(new(Store), new(*MemoryStore)),
)
