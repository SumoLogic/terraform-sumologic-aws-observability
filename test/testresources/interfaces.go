package testresources

import "testing"

// SumoResource is implemented by any Sumo Logic resource a test needs to pre-create.
type SumoResource interface {
	Create(t *testing.T) string
	Delete(t *testing.T)
	ID() string
}

// AWSResource is implemented by any AWS resource a test needs to pre-create.
type AWSResource interface {
	Create(t *testing.T) string
	Delete(t *testing.T)
	ID() string
}
