package list

import (
	"github.com/muesli/reflow/ansi"
	"strings"
	"testing"
)

// test is a shorthand and will be converted to proper testModels
// with embedModels
type test struct {
	vWidth   int
	vHeight  int
	items    []string
	shouldBe string
}

type testModel struct {
	model    Model
	shouldBe string
}

// TestViewBounds is use to make sure that the Renderer String
// NEVER leaves the bounds since then it could mess with the layout.
func TestViewBounds(t *testing.T) {
	for _, testM := range embedModels(genTestModels()) {
		for i, line := range strings.Split(View(testM.model), "\n") {
			lineWidth := ansi.PrintableRuneWidth(line)
			width := testM.model.Viewport.Width
			if lineWidth > width {
				t.Errorf("The line:\n\n%s\n%s^\n\n is %d chars longer than the Viewport width.", line, strings.Repeat(" ", width-1), lineWidth-width)
			}
			if i > testM.model.Viewport.Height {
				t.Error("There are more lines produced from the View() than the Viewport height")
			}
		}
	}
}

// TestGoldenSamples checks the View's string result against a knowen string (golden sample)
// Because there is no margin for diviations, if the test fails, lock also if the "golden sample" is sane.
func TestGoldenSamples(t *testing.T) {
	for _, testM := range embedModels(genTestModels()) {
		actual := View(testM.model)
		expected := testM.shouldBe
		if actual != expected {
			t.Errorf("expected Output:\n\n%s\n\nactual Output:\n\n%s\n\n", expected, actual)
		}
	}
}

// TestPanic is also a golden sampling, but for cases that should panic.
func TestPanic(t *testing.T) {
	for _, testM := range embedModels(genPanicTests()) {
		View(testM)
		actual := recover()
		expected := testM.shouldBe
		if actual != expected {
			t.Errorf("expected panic Output:\n\n%s\n\nactual Output:\n\n%s\n\n", expected, actual)
		}
	}
}

// small helper function to generate simple test cases.
// for more elaborate ones append them afterwards.
func genTestModels() []test {
	return []test{
		// The default has abs linenumber and this seperator enabled
		// so that even if the terminal does not support colors
		// all propertys are still distinguishable.
		{
			240,
			80,
			[]string{
				"",
			},
			"\x1b[7m0 ╭>\x1b[0m\n",
		},
		// if exceding the boards and softwrap (at word bounderys are possible
		// wrap there. Dont increment the item number because its still the same item.
		{
			10,
			2,
			[]string{
				"robert frost",
			},
			"\x1b[7m0 ╭>robert\x1b[0m\n\x1b[7m  │ frost\x1b[0m\n",
		},
	}
}

func embedModels(rawLists []test) []testModel {
	processedList := make([]testModel, len(rawLists))
	for i, list := range rawLists {
		m := NewModel()
		m.Viewport.Height = list.vHeight
		m.Viewport.Width = list.vWidth
		m.AddItems(list.items)
		newItem := testModel{model: m, shouldBe: list.shouldBe}
		processedList[i] = newItem
	}
	return processedList
}

//
func genPanicTests() []test {
	return []test{
		// no width to display -> panic
		{
			0,
			1,
			[]string{""},
			"Can't display with zero width or hight of Viewport",
		},
		// no height to display -> panic
		{
			1,
			0,
			[]string{""},
			"Can't display with zero width or hight of Viewport",
		},
		// no item to display -> panic
		{
			1,
			1,
			[]string{},
			"",
		},
	}
}