package server

import "fmt"

// errRequired returns an error for a missing required tool parameter.
func errRequired(name string) error {
	return fmt.Errorf("required parameter missing: %s", name)
}

// errNotImplemented returns an error for a surface the current client cannot
// perform (reported plainly, never faked).
func errNotImplemented(what string) error {
	return fmt.Errorf("not implemented: %s", what)
}
