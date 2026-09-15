package app

import (
	"raccodown/internal/domain/service/passwordhasher"
	"raccodown/internal/domain/service/tokengenerator"
)

// services bundles the domain services shared across use cases (both the login and
// update-credentials use cases need the hasher, for instance), so each is constructed once here
// rather than once per use case.
type services struct {
	Hasher *passwordhasher.BcryptHasher
	Tokens *tokengenerator.RandomTokenGenerator
}

// newServices constructs the shared domain services. They're all stateless/self-contained, so this
// never fails.
func newServices() services {
	return services{
		Hasher: passwordhasher.New(),
		Tokens: tokengenerator.New(),
	}
}
