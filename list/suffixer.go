package list

import (
	"fmt"
	"github.com/muesli/reflow/ansi"
)

// Suffixer is used to suffix all visible Lines.
// InitSuffixer gets called ones on the beginning of the Lines method
// and then Suffix ones, per line to draw, to generate according suffixes.
type Suffixer interface {
	InitSuffixer(value fmt.Stringer, currentItemIndex, cursorIndex, lineOffset, width, height int) int
	Suffix(currentLine, allLines int) string
}

// DefaultSuffixer is more a example than a default but still it highlights
// the usage and the line. Also if used the line gets padded to the List Width
// So that it can be horizontally joined with other strings/Views.
type DefaultSuffixer struct {
	currentMarker string
	markerLenght  int
	itemIndex     int
	width         int
	cursorIndex   int
}

// NewSuffixer returns a simple suffixer
func NewSuffixer() *DefaultSuffixer {
	return &DefaultSuffixer{currentMarker: "<"}
}

// InitSuffixer returns the visible Width of the strings used to suffix the lines
func (e *DefaultSuffixer) InitSuffixer(_ fmt.Stringer, currentItemIndex, cursorIndex, lineOffset, width, height int) int {
	e.itemIndex = currentItemIndex
	e.width = width
	e.cursorIndex = cursorIndex
	e.markerLenght = ansi.PrintableRuneWidth(e.currentMarker)
	return e.markerLenght
}

// Suffix returns a suffix string for the given line
func (e *DefaultSuffixer) Suffix(line, allLines int) string {
	if e.itemIndex == e.cursorIndex && line == 0 {
		return e.currentMarker
	}
	// a line with a empty suffix string becomes not padded with spaces
	// so if you want to have everything padded to the list-width, return a space.
	return ""
}
