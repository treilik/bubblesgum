package boxer

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/termenv"
)

// Leave are the Boxers which holds the content Models
type Leave struct {
	Content     tea.Model
	Border      bool
	BorderStyle termenv.Style
	Width       int
	Heigth      int
	innerHeigth int
	innerWidth  int

	N, NW, W, SW, S, SO, O, NO string
}

// NewLeave returns a leave with border enabled and set
func NewLeave() Leave {
	return Leave{
		Border: true,
		N:      "─",
		NW:     "╭",
		W:      "│",
		SW:     "╰",
		NO:     "╮",
		O:      "│",
		SO:     "╯",
		S:      "─",
	}
}

// Init is a proxy to the Content Init
func (l Leave) Init() tea.Cmd {
	return l.Content.Init()
}

// InitAll calls the Init of the Content and returns the cmd embedded in a array to satisfiy the Boxer-interface
func (l Leave) InitAll() []tea.Cmd {
	return []tea.Cmd{l.Content.Init()}
}

// Update takes care about the seting of the id of this leave
// and the changing of the WindowSizeMsg depending on the border
// and the focus style of the border.
func (l Leave) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	boxer, cmdList := l.Update(msg)
	return boxer, tea.Batch(cmdList)
}

// UpdateAll calls the Update of the Content and returns the cmd embedded in a array to satisfiy the Boxer-interface
func (l Leave) UpdateAll(msg tea.Msg) (Boxer, []tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.Width = msg.Width
		l.Heigth = msg.Height
		if !l.Border {
			l.innerWidth = msg.Width
			l.innerHeigth = msg.Height
			newContent, cmd := l.Content.Update(msg)
			l.Content = newContent
			return l, []tea.Cmd{cmd}
		}
		// account for Border width/heigth
		l.innerHeigth = msg.Height - strings.Count(l.N+l.S, NEWLINE) - 2
		l.innerWidth = msg.Width - ansi.PrintableRuneWidth(l.W+l.O)
		newContent, cmd := l.Content.Update(tea.WindowSizeMsg{Height: l.innerHeigth, Width: l.innerWidth})
		l.Content = newContent
		return l, []tea.Cmd{cmd}
	}
	newContent, cmd := l.Content.Update(msg)
	l.Content = newContent
	return l, []tea.Cmd{cmd}
}

// View is used to satisfy the tea.Model interface and returnes either the joined lines
// or the Error string if err of lines is not nil.
func (l Leave) View() string {
	lines, err := l.lines()
	if err != nil {
		return err.Error()
	}
	return strings.Join(lines, NEWLINE)
}

// Lines returns the fully rendert (maybe with borders) lines and fullfills the Boxer Interface.
// Error may be of type ProporationError.
func (l Leave) Lines() ([]string, error) {
	return l.lines()
}

func (l *Leave) lines() ([]string, error) {
	boxer, ok := l.Content.(Boxer)
	var lines []string
	var err error
	if ok {
		lines, err = boxer.Lines()
	} else {
		lines = strings.Split(l.Content.View(), NEWLINE)
	}
	if err != nil {
		lines = strings.Split(err.Error(), NEWLINE)
		if len(lines) > l.innerHeigth {
			// limit the lines, so that NewProporationError does not overwrite the original error
			lines = lines[:l.innerHeigth]
		}
	}
	if length := len(lines); length > l.innerHeigth {
		err = NewProporationError(l)
	}

	// expand to match heigth
	if len(lines) < l.innerHeigth {
		lines = append(lines, make([]string, l.innerHeigth-len(lines))...)
	}

	// expand to match width
	for i, line := range lines {
		lineWidth := ansi.PrintableRuneWidth(line)
		if lineWidth > l.innerWidth {
			return nil, NewProporationError(l)
		}
		lines[i] = line + strings.Repeat(SPACE, l.innerWidth-lineWidth)
	}

	if !l.Border {
		return lines, err
	}
	// draw border
	fullLines := make([]string, 0, len(lines)+strings.Count(l.N, NEWLINE)+1)
	firstLine := l.NW + strings.Repeat(l.N, l.innerWidth) + l.NO
	begin, end := l.W, l.O
	lastLine := l.SW + strings.Repeat(l.S, l.innerWidth) + l.SO

	// if set style border
	if l.BorderStyle.String() != "" {
		firstLine = l.BorderStyle.Styled(firstLine)
		begin = l.BorderStyle.Styled(begin)
		end = l.BorderStyle.Styled(end)
		lastLine = l.BorderStyle.Styled(lastLine)
	}

	fullLines = append(fullLines, firstLine)
	for _, line := range lines {
		fullLines = append(fullLines, begin+line+end)
	}
	fullLines = append(fullLines, lastLine)

	return fullLines, err
}
