package boxer

import (
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/ansi"
)

const (
	// SPACE is used to fill up lines
	SPACE = " "
	//NEWLINE is the newline string
	NEWLINE = "\n" // TODO make windows compatible
)

// Boxer is a interface to render multiple bubbles (within a tree) to the terminal screen.
type Boxer interface {
	tea.Model
	InitAll() []tea.Cmd
	UpdateAll(tea.Msg) (Boxer, []tea.Cmd)
	Lines() ([]string, error)
}

// Model is a bubble to manage/bundle other bubbles into boxes on the screen
type Model struct {
	root          bool
	paths         map[string][]nodePos
	children      []BoxSize
	Height, Width int
	Vertical      bool
	lastFocused   int
	Sizer         func(childLenght int, vertical bool, msg tea.WindowSizeMsg) ([]tea.WindowSizeMsg, error)

	requestID chan<- chan int
}

// BoxSize holds a boxer value and the current size the box of this boxer should have
type BoxSize struct {
	Box           Boxer
	Width, Height int
}

type pathInfo struct {
	path    []nodePos
	address string
}

// FocusLeave is used to gather the path of each leave while its transported to the leave.
type FocusLeave struct {
	path           []nodePos
	vertical, next bool
}

// AddressMsg is a Command to update a specific node in the Boxer-tree
type AddressMsg struct {
	path    []nodePos
	Msg     tea.Msg // TODO Change to Cmd?
	Address string
}

// ChangeFocus is the answer of FocusLeave and tells the parents to change the focus of the leaves by two msg.
type ChangeFocus struct {
	newFocus    FocusLeave
	focus       bool
	handledPath []nodePos
}

type nodePos struct {
	index       int
	vertical    bool
	childAmount int
}

// Init call the Init methods of the Children and returns the batched/collected returned Cmd's of them
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.InitAll()...)
}

// InitAll call the Init methods of the Children and returns the batched/collected returned Cmd's of them
func (m Model) InitAll() []tea.Cmd {
	cmdList := make([]tea.Cmd, len(m.children))
	for _, child := range m.children {
		cmdList = append(cmdList, child.Box.InitAll()...)
	}
	return cmdList
}

// Update handles the ratios between the different Boxers
// through the according fanning of the WindowSizeMsg's
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	boxer, cmdList := m.UpdateAll(msg)
	return boxer, tea.Batch(cmdList...)
}

// UpdateAll handles the ratios between the different Boxers
// through the according fanning of the WindowSizeMsg's
func (m Model) UpdateAll(msg tea.Msg) (Boxer, []tea.Cmd) {
	var cmdList []tea.Cmd
	switch msg := msg.(type) {
	case AddressMsg:
		if m.root {
			path, ok := m.paths[msg.Address]
			if !ok {
				return m, toCmdArray(fmt.Errorf("address '%s' not found ", msg.Address))
				// TODO change to own error type
			}
			msg.path = path
		}
		if len(msg.path) == 0 {
			// can't follow path -> return error
			return m, toCmdArray(NewEmptyPath(msg))
		}
		next := msg.path[0]
		if next.childAmount > len(m.children) || next.index > len(m.children) {
			// path does not exists -> return error
			return m, toCmdArray(fmt.Errorf("cant follow path"))
			// TODO change to own error type
		}

		// follow path
		var rest []nodePos
		if len(msg.path) > 0 {
			rest = msg.path[1:]
		}
		msg.path = rest
		newModel, cmd := m.children[next.index].Box.UpdateAll(msg)
		newBox, ok := newModel.(Boxer)
		if !ok {
			return m, toCmdArray(fmt.Errorf("one child returned something else than a boxer: %T", newBox))
			// TODO change to own error type
		}
		m.children[next.index].Box = newBox
		return m, cmd

	case FocusLeave:
		length := len(m.children)
		mu := &sync.Mutex{}
		wg := &sync.WaitGroup{}
		for i, box := range m.children {
			wg.Add(1)
			go func(m *Model, box BoxSize, i int) {
				// for each child append its position to the path
				newMsg := msg
				newMsg.path = append(msg.path, nodePos{index: i, vertical: m.Vertical, childAmount: length})
				newModel, cmd := box.Box.UpdateAll(newMsg)
				// Focus
				newBoxer, ok := newModel.(Boxer)
				if !ok { // TODO
					panic("Not a Boxer")
				}
				box.Box = newBoxer
				mu.Lock()
				m.children[i] = box
				cmdList = append(cmdList, cmd...)
				mu.Unlock()
				wg.Done()
			}(&m, box, i)
		}
		wg.Wait()
		return m, cmdList

	// ChangeFocus is a exception to the FAN-OUT of the Msg's because its follows the specific path defined by the Msg-emitter.
	case ChangeFocus:
		// default to the last focused
		targetIndex := m.lastFocused

		// if path is not empyt
		if len(msg.newFocus.path) > 0 {
			// follow the path
			targetIndex = msg.newFocus.path[0].index
		}

		// path is empty => dont know where to go.
		if len(msg.newFocus.path) == 0 {
			// default to the first in the direction of the movement. (i.e. the first or the last)
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
		newModel, cmd := m.children[targetIndex].Box.UpdateAll(childMsg)
		newBox, ok := newModel.(Boxer)
		if !ok {
			cmd = toCmdArray(NewWrongTypeError(newBox, "boxer.Boxer"))
		}
		m.children[targetIndex].Box = newBox
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, toCmdArray(tea.Quit())
		case "alt+right":
			return m, toCmdArray(FocusLeave{next: true, vertical: false})
		case "alt+left":
			return m, toCmdArray(FocusLeave{next: false, vertical: false})
		case "alt+up":
			return m, toCmdArray(FocusLeave{next: false, vertical: true})
		case "alt+down":
			return m, toCmdArray(FocusLeave{next: true, vertical: true})
		default:
			for i, box := range m.children {
				newModel, cmd := box.Box.UpdateAll(msg)
				newBoxer, ok := newModel.(Boxer)
				if !ok {
					continue
				}
				box.Box = newBoxer
				m.children[i] = box
				cmdList = append(cmdList, cmd...)
			}
		}
		return m, cmdList
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		if m.Sizer != nil {
			newSizes, err := m.Sizer(len(m.children), m.Vertical, msg)
			if err == nil && len(newSizes) == len(m.children) {
				for i, box := range m.children {
					model, cmd := box.Box.UpdateAll(newSizes[i])
					box := model.(Boxer)
					m.children[i].Box = box
					m.children[i].Height = newSizes[i].Height
					m.children[i].Width = newSizes[i].Width
					cmdList = append(cmdList, cmd...)
				}
				return m, cmdList
			}
			cmdList = append(cmdList, toCmdArray(err)...)
		}

		amount := len(m.children)
		quotient := msg.Width / amount
		remainder := msg.Width - quotient*amount
		if m.Vertical {
			quotient = msg.Height / amount
			remainder = msg.Height - quotient*amount
		}
		mu := &sync.Mutex{}
		wg := &sync.WaitGroup{}
		for i, box := range m.children {
			wg.Add(1)
			go func(m *Model, box BoxSize, i int) {
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
				newModel, cmd := box.Box.UpdateAll(tea.WindowSizeMsg{Height: newHeigth, Width: newWidth})
				newBoxer, ok := newModel.(Boxer)
				if !ok {
					panic("not a boxer") // TODO
				}
				box.Box = newBoxer
				box.Height = newHeigth
				box.Width = newWidth
				mu.Lock()
				m.children[i] = box
				cmdList = append(cmdList, cmd...)
				mu.Unlock()
				wg.Done()
			}(&m, box, i)
		}
		wg.Wait()
		return m, cmdList
	default:
		mu := &sync.Mutex{}
		wg := &sync.WaitGroup{}
		for i, box := range m.children {
			wg.Add(1)
			go func(m *Model, box BoxSize, i int) {
				newModel, cmd := box.Box.UpdateAll(msg)
				newBoxer, ok := newModel.(Boxer)
				if ok {
					box.Box = newBoxer
				}
				mu.Lock()
				m.children[i] = box
				cmdList = append(cmdList, cmd...)
				mu.Unlock()
				wg.Done()

			}(&m, box, i)
		}
		wg.Wait()
		return m, cmdList
	}
}

// View is only used for the top (root) node since all other Models use the Lines function.
func (m Model) View() string {
	// The error is ignored here since we can't return it and it would (when printed) overwrite all the boxes.
	lines, _ := m.lines()
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
		err := NewNoChildrenError()
		return strings.Split(err.Error(), NEWLINE), err
	}
	//            y  x
	var joinedStr [][]string
	targetHeigth := m.Height
	var errList MultipleErrors
	// bring all to same height if they are smaller
	for _, boxer := range m.children {
		lines, err := boxer.Box.Lines()
		if err != nil {
			errList = append(errList, err)
		}

		if targetHeigth > boxer.Height {
			err := NewWrongSizeError(0, targetHeigth, 0, boxer.Height)
			lines = strings.Split(err.Error(), NEWLINE)
		}
		if len(lines) < boxer.Height {
			lines = append(lines, make([]string, boxer.Height-len(lines))...)
		}
		joinedStr = append(joinedStr, lines)
		targetHeigth = boxer.Height
	}

	lenght := len(joinedStr)
	// Join the horizontal lines together
	var allStr []string
	// y
	for c := 0; c < targetHeigth; c++ {
		fullLine := make([]string, 0, lenght)
		// x
		for i := 0; i < lenght; i++ {
			boxWidth := m.children[i].Width
			line := joinedStr[i][c]
			lineWidth := ansi.PrintableRuneWidth(line)
			if lineWidth > boxWidth {
				err := NewProporationError(m.children[i].Box)
				allStr = strings.Split(err.Error(), NEWLINE)
			}
			var pad string
			if lineWidth < boxWidth {
				pad = strings.Repeat(SPACE, boxWidth-lineWidth)
			}
			fullLine = append(fullLine, line, pad)
		}
		allStr = append(allStr, strings.Join(fullLine, ""))
	}

	return allStr, errList
}

func (m *Model) upDownJoin() ([]string, error) {

	if len(m.children) == 0 {
		err := NewNoChildrenError()
		return strings.Split(err.Error(), NEWLINE), err
	}
	boxWidth := m.children[0].Width
	boxes := make([]string, 0, m.Height)
	targetWidth := m.Width
	var errList MultipleErrors
	for _, child := range m.children {
		if child.Box == nil {
			err := NewNoChildrenError()
			return strings.Split(err.Error(), NEWLINE), err
		}
		lines, err := child.Box.Lines()
		if err != nil {
			errList = append(errList, err)
		}
		if len(lines) > child.Height {
			err := NewProporationError(child.Box)
			lines = strings.Split(err.Error(), NEWLINE)
		}
		// check for to wide lines and because we are on it, pad them to correct width.
		for _, line := range lines {
			lineWidth := ansi.PrintableRuneWidth(line)
			if lineWidth != targetWidth {
				err := NewWrongSizeError(lineWidth, 0, targetWidth, 0)
				if err != nil {
					line = err.Error() // TODO change error handling?
				}
				lineWidth = ansi.PrintableRuneWidth(line)
				if lineWidth > targetWidth {
					line = line[:targetWidth] // TODO handle ansi better
					lineWidth = ansi.PrintableRuneWidth(line)
				}
			}
			line += strings.Repeat(SPACE, boxWidth-lineWidth)
		}
		boxes = append(boxes, lines...)
		// add more lines to boxes to match the Height of the child-box
		for c := 0; c < child.Height-len(lines); c++ {
			boxes = append(boxes, strings.Repeat(SPACE, boxWidth))
		}
	}
	return boxes, errList
}

// AddChildren adds the given BoxerSize's as children
// If one provided BoxerSize.Box is 'nil' an NotABoxerError is returned
// and no child is added!
func (m *Model) AddChildren(cList ...BoxSize) error {
	newChildren := make([]BoxSize, 0, len(cList))
	for _, newChild := range cList {
		switch c := newChild.Box.(type) {
		case Model:
			c.requestID = m.requestID
			newChild.Box = c
			newChildren = append(newChildren, newChild)
		case nil:
			return NewNotABoxerError(c)
		default:
			newChild.Box = c
			newChildren = append(newChildren, newChild)
		}
	}
	m.children = append(m.children, newChildren...)
	return nil
}

// getID returns a new for this Model(-tree) unique id
// to identify the nodes/leave and direct the message flow.
func (m *Model) getID() int {
	if m.requestID == nil {
		req := make(chan chan int)

		m.requestID = req

		// the id '0' is skipped to be able to distinguish zero-value and proper id TODO is this a valid/good way to go?
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

func toCmdArray(msgList ...tea.Msg) []tea.Cmd {
	cmdList := make([]tea.Cmd, 0, len(msgList))
	for _, msg := range msgList {
		if msg == nil {
			continue
		}
		cmdList = append(cmdList, func() tea.Msg { return msg })
	}
	return cmdList
}
