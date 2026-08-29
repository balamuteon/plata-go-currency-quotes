package quote

import (
	"context"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=quote_test

type processor interface {
	ProcessNext(ctx context.Context) (bool, error)
}
