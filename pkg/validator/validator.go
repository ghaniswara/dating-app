package validator

import "context"

type Validate interface {
	Validate(ctx context.Context) (problems map[string]string)
}
