package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/treilik/bubblesgum/boxer"
	"github.com/treilik/bubblesgum/list"
)

func main() {
	leftList := list.NewModel()
	leftLeave := boxer.NewLeave()
	leftLeave.Content = leftList
	leftLeave.Address = "left"

	middleList := list.NewModel()
	middleLeave := boxer.NewLeave()
	middleLeave.Content = middleList
	middleLeave.Address = "middle"
	middleLeave.Focus = true

	rigthList := list.NewModel()
	rigthLeave := boxer.NewLeave()
	rigthLeave.Content = rigthList
	rigthLeave.Address = "right"

	contentNode := boxer.Model{}
	contentNode.AddChildren(
		boxer.BoxSize{Box: leftLeave},
		boxer.BoxSize{Box: middleLeave},
		boxer.BoxSize{Box: rigthLeave},
	)
	contentNode.Sizer = func(childernLen int, vertical bool, msg tea.WindowSizeMsg) ([]tea.WindowSizeMsg, error) {
		first := msg.Width / 6
		second := msg.Width / 3
		third := msg.Width - first - second
		return []tea.WindowSizeMsg{
			{Width: first, Height: msg.Height},
			{Width: second, Height: msg.Height},
			{Width: third, Height: msg.Height},
		}, nil
	}

	errorList := list.NewModel()
	errorLeave := boxer.NewLeave()
	errorLeave.Content = errorList
	errorLeave.Address = "error"

	boxerRoot := boxer.Model{}
	boxerRoot.Vertical = true
	boxerRoot.AddChildren(
		boxer.BoxSize{Box: contentNode},
		boxer.BoxSize{Box: errorLeave},
	)
	boxerRoot.Sizer = func(childernLen int, vertical bool, msg tea.WindowSizeMsg) ([]tea.WindowSizeMsg, error) {
		if msg.Height < 10 {
			return nil, fmt.Errorf("too less lines for custom size for children")
		}
		if childernLen != 2 {
			return nil, fmt.Errorf("wrong amount of children for this size layout")
		}
		errorHight := 4

		return []tea.WindowSizeMsg{
			{Width: msg.Width, Height: msg.Height - errorHight},
			{Width: msg.Width, Height: errorHight},
		}, nil
	}

	grandChildNode := node{}
	grandChildNode.value = "grandchild"

	leftChildNode := node{}
	leftChildNode.value = "leftChild"
	leftChildNode.children = []stringerTree{grandChildNode}

	rightChildNode := node{}
	rightChildNode.value = "rightChild"

	rootNode := node{}
	rootNode.value = "root"
	rootNode.children = []stringerTree{leftChildNode, rightChildNode}

	root := treeViewer{}
	root.boxView = boxerRoot
	root.tree = rootNode
	p := tea.NewProgram(root)
	p.EnterAltScreen()
	if err := p.Start(); err != nil {
		fmt.Println("could not start program")
		os.Exit(1)
	}
	p.ExitAltScreen()
}

type treeViewer struct {
	boxView boxer.Boxer
	tree    stringerTree
	ready   bool
	errList []fmt.Stringer
}

func (t treeViewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !t.ready {
		_, ok := msg.(boxer.Ready)
		if !ok {
			// pipe all msgs through which boxer needs to be ready
			newModel, cmd := t.boxView.Update(msg)
			newBox := newModel.(boxer.Boxer)
			t.boxView = newBox
			return t, cmd
		}
		t.ready = true
	}
	switch msg := msg.(type) {
	case boxer.Ready:
		leftCmd := boxer.AddressMsg{Address: "left", Msg: list.ResetItems{t.tree}}
		newModel, leftListCmd := t.boxView.Update(leftCmd)
		newBox := newModel.(boxer.Boxer)
		t.boxView = newBox

		middleChildren := t.tree.Children()
		var middleList list.ResetItems
		for _, child := range middleChildren {
			middleList = append(middleList, child)
		}
		if len(middleList) == 0 {
			return t, leftListCmd
		}
		middleCmd := boxer.AddressMsg{Address: "middle", Msg: middleList}
		newModel, middleListCmd := t.boxView.Update(middleCmd)
		newBox = newModel.(boxer.Boxer)
		t.boxView = newBox

		if len(middleChildren) == 0 {
			return t, tea.Batch(leftListCmd, middleListCmd)
		}
		rightChildren := middleChildren[0].Children()
		var rightList list.ResetItems
		for _, child := range rightChildren {
			rightList = append(rightList, child)
		}
		rightCmd := boxer.AddressMsg{Address: "right", Msg: rightList}
		newModel, rightListCmd := t.boxView.Update(rightCmd)
		newBox = newModel.(boxer.Boxer)
		t.boxView = newBox

		return t, tea.Batch(leftListCmd, middleListCmd, rightListCmd)
	case error:
		t.errList = append(t.errList, errorStringer{msg})
		reversed := make([]fmt.Stringer, 0, len(t.errList))
		for c := len(t.errList) - 1; c >= 0; c-- {
			reversed = append(reversed, t.errList[c])
		}
		errorCmd := boxer.AddressMsg{Address: "error", Msg: list.ResetItems(reversed)}
		newModel, cmd := t.boxView.Update(errorCmd)
		newBox := newModel.(boxer.Boxer)
		t.boxView = newBox

		return t, cmd

	default:
		newModel, cmd := t.boxView.Update(msg)
		newBox := newModel.(boxer.Boxer)
		t.boxView = newBox
		return t, cmd
	}
}
func (t treeViewer) Init() tea.Cmd            { return t.boxView.Init() }
func (t treeViewer) View() string             { return t.boxView.View() }
func (t treeViewer) Lines() ([]string, error) { return t.boxView.Lines() }

type stringerTree interface {
	String() string
	Children() []stringerTree
}

type node struct {
	value    string
	children []stringerTree
}

func (n node) String() string {
	return n.value
}
func (n node) Children() []stringerTree {
	return n.children
}

type errorStringer struct {
	err error
}

func (e errorStringer) String() string {
	return e.err.Error()
}
