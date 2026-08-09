package server

import "fmt"

// errRequired returns an error for a missing required tool parameter.
func errRequired(name string) error {
	return fmt.Errorf("required parameter missing: %s", name)
}

// errUnsupported makes an advertised input explicit when the pinned Incus
// client and target API have no supported representation for it.
func errUnsupported(name string) error {
	return fmt.Errorf("unsupported parameter: %s", name)
}
