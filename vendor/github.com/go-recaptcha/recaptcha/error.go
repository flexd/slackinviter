package recaptcha

import "fmt"

type Error struct {
	Codes []string
}

func (e *Error) Error() string {
	return fmt.Sprintf("validation failed: %v", e.Codes)
}
