package types

import "fmt"

// Endpoint represents a network endpoint with IP and port
type Endpoint struct {
	IP   string
	Port int
}

// String returns a string representation of the endpoint
func (e Endpoint) String() string {
	return fmt.Sprintf("%s:%d", e.IP, e.Port)
}

// STUNError represents an error during STUN operations
type STUNError struct {
	Op  string // Operation that failed
	Err error  // Underlying error
}

func (e *STUNError) Error() string {
	return fmt.Sprintf("STUN %s: %v", e.Op, e.Err)
}

func (e *STUNError) Unwrap() error {
	return e.Err
}

// NewSTUNError creates a new STUN error
func NewSTUNError(op string, err error) error {
	return &STUNError{
		Op:  op,
		Err: err,
	}
}
