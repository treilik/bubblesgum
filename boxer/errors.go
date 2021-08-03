package boxer

import (
	"fmt"
	"strings"
)

// ProportionError is for signaling that the string return by the View or Lines function has wrong proportions(width/height)
type ProportionError error

// NewProporationError returns a uniform string for this error
func NewProporationError(b Boxer) error {
	return ProportionError(fmt.Errorf("the Lines function of this boxer: '%v'%shas returned to much or long lines", b, NEWLINE))
}

// NoChildrenError is an error used to signal that a Boxer can not render itself when it has no Children to render.
type NoChildrenError error

// NewNoChildrenError returns a NoChildrenError
func NewNoChildrenError() error {
	return NoChildrenError(fmt.Errorf("no Children to get lines from"))
}

// WrongSizeError shall be used to signal that a string exceeds the necessary Width or hight.
type WrongSizeError error

// NewWrongSizeError returns a WrongSizeError if the is[Width/Height] is grater then the corresponding shall[Width/Height] other nil is returned.
func NewWrongSizeError(isWidth, isHeight, shallWidth, shallHeight int) error {
	if isWidth > shallWidth && isHeight > shallHeight {
		return WrongSizeError(fmt.Errorf("the given Height: %d and Width: %d are to big for the possible Height: %d adn width: %d", isHeight, isWidth, shallHeight, shallWidth))
	}
	if isWidth > shallWidth {
		return WrongSizeError(fmt.Errorf("the given Width: %d is to big for the possible Width: %d", isWidth, shallWidth))
	}
	if isHeight > shallHeight {
		return WrongSizeError(fmt.Errorf("the given Height: %d is to big for the possible Height: %d", isHeight, shallHeight))
	}
	return nil
}

// NotABoxerError is used when something should satisfy the Boxer interface but does not (like 'nil').
type NotABoxerError error

// NewNotABoxerError returns a new NotABoxerError.
func NewNotABoxerError(b Boxer) error {
	return NotABoxerError(fmt.Errorf("%q is not a Boxer", b))
}

// MultipleErrors is used to collect multiple errors from different sources
type MultipleErrors []error

func (me MultipleErrors) Error() string {
	allErr := []string{"there where multiple errors:"}
	for i, err := range me {
		allErr = append(allErr, fmt.Sprintf("%d > %q", i, err.Error()))
	}
	return strings.Join(allErr, NEWLINE)
}

// WrongTypeError is used to show when the type provided does not match any of the acceptable types.
type WrongTypeError error

// NewWrongTypeError returns a new WrongTypeError
func NewWrongTypeError(got interface{}, want ...string) error {
	return WrongTypeError(fmt.Errorf("%v entry could not be added, because the type does not match something of %q.\n Most likely you have to embed your model in a 'boxer.Leave.Content'", got, want))
}
