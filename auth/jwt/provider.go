package jwt

import "github.com/google/wire"

// ProviderSet is the Wire provider set for the JWT authenticator.
var ProviderSet = wire.NewSet(New)
