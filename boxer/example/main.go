package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/treilik/bubblesgum/boxer"
	"github.com/treilik/bubblesgum/list"
)

func main() {
	lowestFirst := list.NewModel()
	lowestFirst.AddItems(list.MakeStringerList("first", "lowest first child"))
	lowestFirstLeave := boxer.NewLeave()
	lowestFirstLeave.Content = lowestFirst
	lowestFirstLeave.Address = "target"

	lowestSecond := custom{model: list.NewModel()}
	lowestSecond.model.AddItems(list.MakeStringerList("first", "lowest second child test"))
	lowestSecondLeave := boxer.NewLeave()
	lowestSecondLeave.Content = lowestSecond

	grandNode := boxer.Model{}
	grandNode.Vertical = true
	grandNode.AddChildren(boxer.BoxSize{Box: lowestFirstLeave}, boxer.BoxSize{Box: lowestSecondLeave})

	grandChild := list.NewModel()
	grandChild.AddItems(list.MakeStringerList("second", "grandchild"))
	grandLeave := boxer.NewLeave()
	grandLeave.Content = grandChild
	grandLeave.Focus = true

	rightChild := boxer.Model{}
	rightChild.Vertical = true
	rightChild.AddChildren(boxer.BoxSize{Box: grandNode}, boxer.BoxSize{Box: grandLeave})

	leftList := list.NewModel()
	leftList.AddItems(list.MakeStringerList("leftList", "rootchild"))
	leftLeave := boxer.NewLeave()
	leftLeave.Content = leftList

	rigthList := list.NewModel()
	rigthList.AddItems(list.MakeStringerList("rigthList", "rootchild"))
	rigthLeave := boxer.NewLeave()
	rigthLeave.Content = rigthList

	root := boxer.Model{}
	root.AddChildren(
		boxer.BoxSize{Box: rightChild},
		boxer.BoxSize{Box: leftLeave},
		boxer.BoxSize{Box: rigthLeave},
	)
	p := tea.NewProgram(root)
	if err := p.Start(); err != nil {
		fmt.Println("could not start program")
		os.Exit(1)
	}
}

type custom struct {
	model list.Model
	count int
}

func (c custom) Init() tea.Cmd {
	return c.model.Init()
}
func (c custom) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "end" {
		return c, func() tea.Msg {
			return boxer.AddressMsg{Address: "target", Msg: msg}
		}
	}
	m, cmd := c.model.Update(msg)
	l, _ := m.(list.Model)
	c.model = l
	return c, cmd
}

func (c custom) View() string {
	return c.model.View()
}

func (c custom) Lines() ([]string, error) {
	return c.model.Lines()
}
