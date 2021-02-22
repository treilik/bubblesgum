package boxer

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/ansi"
)

const (
	// SPACE is used to fill up lines
	SPACE = "."
	//NEWLINE is the newline string
	NEWLINE = "\n" // TODO make windows compatible
)

// Boxer is a interface to render multiple bubbles (within a tree) to the terminal screen.
type Boxer interface {
	Lines() ([]string, error)
	tea.Model // TODO remove View
}

// Model is a bubble to manage/bundle other bubbles into boxes on the screen
type Model struct {
	children      []BoxSize
	Height, Width int
	Vertical      bool
	id            int
	lastFocused   int

	requestID chan<- chan int
}

// BoxSize holds a boxer value and the current size the box of this boxer should have
type BoxSize struct {
	Box           Boxer
	Width, Heigth int
}

// Start is a Msg to start the id spreading
type Start struct{}

// InitIDs is a Msg to spread the id's of the leaves
type InitIDs chan<- chan int

// ProportionError is for signaling that the string return by the View or Lines function has wrong proportions(width/height)
type ProportionError error

// FocusLeave is used to gather the path of each leave while its trasported to the leave.
type FocusLeave struct {
	path           []nodePos
	vertical, next bool
}

// ChangeFocus is the answere of FocusLeave and tells the parents to change the focus of the leaves by two msg.
type ChangeFocus struct {
	newFocus    FocusLeave
	focus       bool
	handledPath []nodePos
}

type nodePos struct {
	index       int
	vertical    bool
	id          int //TODO remove
	childAmount int
}

// NewProporationError returns a uniform string for this error
func NewProporationError(b Boxer) error {
	return fmt.Errorf("the Lines function of this boxer: '%v'%shas returned to much or long lines", b, NEWLINE)
}

// Init call the Init methodes of the Children and returns the batched/collected returned Cmd's of them
func (m Model) Init() tea.Cmd {
	cmdList := make([]tea.Cmd, len(m.children))
	for _, child := range m.children {
		cmdList = append(cmdList, child.Box.Init())
	}
	// the adding of the Start Msg leads to multiple Msg while only one is used and the rest gets ignored
	cmdList = append(cmdList, func() tea.Msg { return Start{} })
	return tea.Batch(cmdList...)
}

// Update handles the ratios between the different Boxers
// through the according fanning of the WindowSizeMsg's
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdList []tea.Cmd
	switch msg := msg.(type) {
	case Start:
		// only the root node gets this all other ids will be set through the spreading of InitIDs
		// TODO should root node be a own struct? to handel the id spread-starting cleaner.
		if m.requestID != nil {
			return m, nil
		}
		m.id = m.getID()
		return m, func() tea.Msg { return InitIDs(m.requestID) }

	case InitIDs:
		if m.requestID == nil {
			m.requestID = msg
			genID := make(chan int)
			m.requestID <- genID
			m.id = <-genID
		}
		for i, box := range m.children {
			newModel, cmd := box.Box.Update(msg)
			newBoxer, ok := newModel.(Boxer)
			if !ok {
				continue // TODO dont ignore this error
			}
			box.Box = newBoxer
			m.children[i] = box
			cmdList = append(cmdList, cmd)
		}
		return m, tea.Batch(cmdList...)

	// FocusLeave is a exception to the FAN-OUT of the Msg's because for each child there is a specific msg, similar to the WindowSizeMsg.
	case FocusLeave:
		length := len(m.children)
		for i, box := range m.children {
			// for each child append its position to the path
			newMsg := msg
			newMsg.path = append(msg.path, nodePos{index: i, vertical: m.Vertical, id: m.id, childAmount: length})
			newModel, cmd := box.Box.Update(newMsg)
			// Focus
			newBoxer, ok := newModel.(Boxer)
			if !ok { // TODO
				continue
			}
			box.Box = newBoxer
			m.children[i] = box
			cmdList = append(cmdList, cmd)
		}
		return m, tea.Batch(cmdList...)

	// ChangedFocus is a exception to the FAN-OUT of the Msg's because its follows the specific path defined by the Msg-emitter.
	case ChangeFocus:
		// default to the last focused
		targetIndex := m.lastFocused

		// path is not empyt
		if len(msg.newFocus.path) > 0 {
			// follow the path
			targetIndex = msg.newFocus.path[0].index
		}

		// path is empty => dont know where to go.
		if len(msg.newFocus.path) == 0 {
			// default to the first in the directon of the movement. (i.e. the first or the last)
			if !msg.newFocus.next && msg.newFocus.vertical == m.Vertical {
				targetIndex = len(m.children) - 1
			}
		}
		// if its not possible to follow the path:
		if targetIndex < 0 || targetIndex >= len(m.children) {
			panic("tree has changed since ChangeFocus was send") // TODO change to error type
		}

		childMsg := ChangeFocus{focus: msg.focus, newFocus: FocusLeave{vertical: msg.newFocus.vertical}}
		if len(msg.newFocus.path) > 0 {
			childMsg.handledPath = append(msg.handledPath, msg.newFocus.path[0])
		}
		if len(msg.newFocus.path) > 1 {
			childMsg.newFocus.path = msg.newFocus.path[1:]
		}
		m.lastFocused = targetIndex
		newModel, cmd := m.children[targetIndex].Box.Update(childMsg)
		var ok bool
		m.children[targetIndex].Box, ok = newModel.(Boxer)
		if !ok {
			panic("wrong type") // TODO
		}
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "alt+right":
			return m, func() tea.Msg { return FocusLeave{next: true, vertical: false} }
		case "alt+left":
			return m, func() tea.Msg { return FocusLeave{next: false, vertical: false} }
		case "alt+up":
			return m, func() tea.Msg { return FocusLeave{next: false, vertical: true} }
		case "alt+down":
			return m, func() tea.Msg { return FocusLeave{next: true, vertical: true} }
		default:
			for i, box := range m.children {
				newModel, cmd := box.Box.Update(msg)
				newBoxer, ok := newModel.(Boxer)
				if !ok {
					continue
				}
				box.Box = newBoxer
				m.children[i] = box
				cmdList = append(cmdList, cmd)
			}
		}
		return m, tea.Batch(cmdList...)
	case tea.WindowSizeMsg:

		amount := len(m.children)
		quotient := msg.Width / amount
		remainder := msg.Width - quotient*amount
		if m.Vertical {
			quotient = msg.Height / amount
			remainder = msg.Height - quotient*amount
		}
		for i, box := range m.children {
			newHeigth := msg.Height
			newWidth := quotient
			if !m.Vertical && remainder > 0 {
				remainder--
				newWidth++
			}
			if m.Vertical {
				newHeigth = quotient
				newWidth = msg.Width
				if remainder > 0 {
					remainder--
					newHeigth++
				}
			}
			newModel, cmd := box.Box.Update(tea.WindowSizeMsg{Height: newHeigth, Width: newWidth})
			newBoxer, ok := newModel.(Boxer)
			if !ok {
				continue // TODO
			}
			box.Box = newBoxer
			box.Heigth = newHeigth
			box.Width = newWidth
			m.children[i] = box
			cmdList = append(cmdList, cmd)
		}
		return m, tea.Batch(cmdList...)
	default:
		for i, box := range m.children {
			newModel, cmd := box.Box.Update(msg)
			newBoxer, ok := newModel.(Boxer)
			if ok {
				box.Box = newBoxer
			}
			m.children[i] = box
			cmdList = append(cmdList, cmd)
		}
		return m, tea.Batch(cmdList...)
	}
}

// View is only used for the top (root) node since all other Models use the Lines function.
func (m Model) View() string {
	lines, err := m.lines()
	if err != nil {
		return err.Error()
	}
	return strings.Join(lines, NEWLINE)
}

// Lines returns the joined lines of all the contained Boxers
func (m Model) Lines() ([]string, error) {
	return m.lines()
}

// Lines returns the joined lines of all the contained Boxers
func (m *Model) lines() ([]string, error) {
	if m.Vertical {
		return m.upDownJoin()
	}
	return m.leftRightJoin()
}

func (m *Model) leftRightJoin() ([]string, error) {
	if len(m.children) == 0 {
		return nil, fmt.Errorf("no children to get lines from")
	}
	//            y  x
	var joinedStr [][]string
	var formerHeigth int
	// bring all to same heigth if they are smaller
	for _, boxer := range m.children {
		lines, err := boxer.Box.Lines()
		if err != nil {
			return nil, err
		}

		if len(lines) < boxer.Heigth {
			lines = append(lines, make([]string, boxer.Heigth-len(lines))...)
		}
		joinedStr = append(joinedStr, lines)
		if formerHeigth > 0 && formerHeigth != boxer.Heigth {
			return nil, fmt.Errorf("for horizontal join all have to be the same heigth") // TODO change to own error type
		}
		formerHeigth = boxer.Heigth
	}

	lenght := len(joinedStr)
	// Join the horizontal lines together
	var allStr []string
	// y
	for c := 0; c < formerHeigth; c++ {
		fullLine := make([]string, 0, lenght)
		// x
		for i := 0; i < lenght; i++ {
			boxWidth := m.children[i].Width
			line := joinedStr[i][c]
			lineWidth := ansi.PrintableRuneWidth(line)
			if lineWidth > boxWidth {
				return nil, NewProporationError(m.children[i].Box)
			}
			var pad string
			if lineWidth < boxWidth {
				pad = strings.Repeat(SPACE, boxWidth-lineWidth)
			}
			fullLine = append(fullLine, line, pad)
		}
		allStr = append(allStr, strings.Join(fullLine, ""))
	}

	return allStr, nil
}

func (m *Model) upDownJoin() ([]string, error) {
	if len(m.children) == 0 {
		return nil, fmt.Errorf("")
	}
	boxWidth := m.children[0].Width
	var boxes []string
	var formerWidth int
	for _, child := range m.children {
		if child.Box == nil {
			return nil, fmt.Errorf("cant work on nil Boxer") // TODO
		}
		lines, err := child.Box.Lines()
		if err != nil {
			return nil, err // TODO limit propagation of errors
		}
		if len(lines) > child.Heigth {
			return nil, NewProporationError(child.Box)
		}
		// check for to wide lines and because we are on it, pad them to corrct width.
		for _, line := range lines {
			lineWidth := ansi.PrintableRuneWidth(line)
			if formerWidth > 0 && lineWidth != formerWidth {
				return nil, fmt.Errorf("for vertical join all boxes have to be the same width") // TODO change to own error type
			}
			line += strings.Repeat(SPACE, boxWidth-lineWidth)
		}
		boxes = append(boxes, lines...)
		// add more lines to boxes to match the Height of the child-box
		for c := 0; c < child.Heigth-len(lines); c++ {
			boxes = append(boxes, strings.Repeat(SPACE, boxWidth))
		}
	}
	return boxes, nil
}

// AddChildren addes the given BoxerSize's as children
// but excludes nil-values and returns after adding the rest a Nil Error
func (m *Model) AddChildren(cList []BoxSize) error {
	var errCount int
	newChildren := make([]BoxSize, 0, len(cList))
	for _, newChild := range cList {
		switch c := newChild.Box.(type) {
		case Model:
			c.requestID = m.requestID
			newChild.Box = c
			newChildren = append(newChildren, newChild)
		case Leave:
			newChild.Box = c
			newChildren = append(newChildren, newChild)
		default:
			errCount++
		}
	}
	m.children = append(m.children, newChildren...)
	if errCount > 0 {
		return fmt.Errorf("%d entrys could not be added", errCount)
	}
	return nil
}

// getID returns a new for this Model(-tree) unique id
// to identify the nodes/leave and direct the message flow.
func (m *Model) getID() int {
	if m.requestID == nil {
		req := make(chan chan int)

		m.requestID = req

		// the id '0' is skiped to be able to distinguish zero-value and proper id TODO is this a valid/good way to go?
		go func(requ <-chan chan int) {
			for c := 2; true; c++ {
				send := <-requ
				send <- c
				close(send)
			}
		}(req)

		return 1
	}
	idChan := make(chan int)
	m.requestID <- idChan
	return <-idChan
}

// func resize(newSize tea.WindowSizeMsg, childrenAmount int) ([]int, error)
