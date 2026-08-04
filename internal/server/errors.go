package server

import "fmt"

// errRequired returns an error for a missing required tool parameter.
func errRequired(name string) error {
	return fmt.Errorf("required parameter missing: %s", name)
}
