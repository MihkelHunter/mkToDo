package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/MihkelHunter/mkToDo/internal/store"
	"github.com/MihkelHunter/mkToDo/internal/todo"
)

// ── Colour palette ───────────────────────────────────────────────────────────

var (
	colBackground = color.NRGBA{R: 15, G: 15, B: 20, A: 255}
	colSurface    = color.NRGBA{R: 26, G: 26, B: 36, A: 255}
	colAccent     = color.NRGBA{R: 99, G: 102, B: 241, A: 255}
	colHighPri    = color.NRGBA{R: 239, G: 68, B: 68, A: 255}
	colMedPri     = color.NRGBA{R: 245, G: 158, B: 11, A: 255}
	colLowPri     = color.NRGBA{R: 100, G: 116, B: 139, A: 255}
)

// ── App state ────────────────────────────────────────────────────────────────

type appState struct {
	svc        *todo.Service
	win        fyne.Window
	taskList   *widget.List
	statsLabel *widget.Label
	tasks      []*todo.Task
	filter     string // "all" | "active" | "done"
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dbDir := filepath.Join(home, ".todoapp")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Fatal(err)
	}

	st, err := store.New(filepath.Join(dbDir, "tasks.db"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}

	svc := todo.NewService(st)
	defer svc.Close()

	a := app.New()
	a.Settings().SetTheme(&darkTheme{})

	win := a.NewWindow("mkToDo")
	win.Resize(fyne.NewSize(740, 600))
	win.CenterOnScreen()

	s := &appState{svc: svc, win: win, filter: "all"}
	win.SetContent(s.buildUI())
	s.refresh()

	win.ShowAndRun()
}

// ── Build UI ─────────────────────────────────────────────────────────────────

func (s *appState) buildUI() fyne.CanvasObject {
	// Header
	title := canvas.NewText("  ✓  TODOApp", color.White)
	title.TextSize = 20
	title.TextStyle = fyne.TextStyle{Bold: true}

	addBtn := widget.NewButton("+ Add Task", func() { s.showTaskForm(nil) })
	addBtn.Importance = widget.HighImportance

	header := container.NewBorder(nil, nil, title, container.NewPadded(addBtn))
	headerBG := canvas.NewRectangle(colSurface)
	headerStack := container.NewStack(headerBG, container.NewPadded(header))

	// Filter tabs
	allBtn := widget.NewButton("All", func() { s.filter = "all"; s.refresh() })
	activeBtn := widget.NewButton("Active", func() { s.filter = "active"; s.refresh() })
	doneBtn := widget.NewButton("Done", func() { s.filter = "done"; s.refresh() })
	filterRow := container.NewHBox(layout.NewSpacer(), allBtn, activeBtn, doneBtn, layout.NewSpacer())

	// Task list
	s.taskList = widget.NewList(
		func() int { return len(s.tasks) },
		s.makeTaskRow,
		s.updateTaskRow,
	)
	s.taskList.OnSelected = func(id widget.ListItemID) { s.taskList.Unselect(id) }

	// Footer / stats
	s.statsLabel = widget.NewLabel("")
	footerBG := canvas.NewRectangle(colSurface)
	footerStack := container.NewStack(footerBG, container.NewPadded(container.NewCenter(s.statsLabel)))

	// Root layout
	bg := canvas.NewRectangle(colBackground)
	ui := container.NewBorder(
		container.NewVBox(headerStack, filterRow),
		footerStack,
		nil, nil,
		container.NewScroll(s.taskList),
	)
	return container.NewStack(bg, ui)
}

// ── Task row template ─────────────────────────────────────────────────────────

func (s *appState) makeTaskRow() fyne.CanvasObject {
	priDot := canvas.NewCircle(colLowPri)
	// priDot.SetMinSize(fyne.NewSize(12, 12))
	priDot.Resize(fyne.NewSize(12, 12))

	checkBtn := widget.NewButtonWithIcon("", theme.RadioButtonIcon(), func() {})
	checkBtn.Importance = widget.LowImportance

	titleLabel := widget.NewLabel("title")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	descLabel := widget.NewLabel("desc")

	editBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {})
	editBtn.Importance = widget.LowImportance

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {})
	deleteBtn.Importance = widget.DangerImportance

	left := container.NewHBox(
		container.NewCenter(priDot),
		checkBtn,
		container.NewVBox(titleLabel, descLabel),
	)
	right := container.NewHBox(editBtn, deleteBtn)
	rowContent := container.NewBorder(nil, nil, left, right)

	rowBG := canvas.NewRectangle(colSurface)
	rowBG.CornerRadius = 8

	return container.NewStack(rowBG, container.NewPadded(rowContent))
}

func (s *appState) updateTaskRow(i widget.ListItemID, obj fyne.CanvasObject) {
	if i >= len(s.tasks) {
		return
	}
	t := s.tasks[i]

	stack := obj.(*fyne.Container)
	rowBG := stack.Objects[0].(*canvas.Rectangle)
	padded := stack.Objects[1].(*fyne.Container)
	border := padded.Objects[0].(*fyne.Container)

	// container.NewBorder stores children as: [top, bottom, left, right, center...]
	// With only left & right set (nil top/bottom), indices: 0=top(nil),1=bottom(nil),2=left,3=right
	left := border.Objects[0].(*fyne.Container)
	right := border.Objects[1].(*fyne.Container)

	priDotBox := left.Objects[0].(*fyne.Container)
	priDot := priDotBox.Objects[0].(*canvas.Circle)
	checkBtn := left.Objects[1].(*widget.Button)
	textBox := left.Objects[2].(*fyne.Container)
	titleLabel := textBox.Objects[0].(*widget.Label)
	descLabel := textBox.Objects[1].(*widget.Label)

	editBtn := right.Objects[0].(*widget.Button)
	deleteBtn := right.Objects[1].(*widget.Button)

	// Priority dot colour
	switch t.Priority {
	case todo.PriorityHigh:
		priDot.FillColor = colHighPri
	case todo.PriorityMedium:
		priDot.FillColor = colMedPri
	default:
		priDot.FillColor = colLowPri
	}
	priDot.Refresh()

	// Done state
	if t.Done {
		checkBtn.SetIcon(theme.ConfirmIcon())
		titleLabel.TextStyle = fyne.TextStyle{Italic: true}
		rowBG.FillColor = color.NRGBA{R: 20, G: 30, B: 25, A: 255}
	} else {
		checkBtn.SetIcon(theme.RadioButtonIcon())
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		rowBG.FillColor = colSurface
	}
	rowBG.Refresh()

	titleLabel.SetText(t.Title)
	if t.Description != "" {
		descLabel.SetText(t.Description)
	} else {
		descLabel.SetText(t.Priority.String() + " priority · " + t.CreatedAt.Format("Jan 2"))
	}

	task := t
	checkBtn.OnTapped = func() { s.toggleTask(task) }
	editBtn.OnTapped = func() { s.showTaskForm(task) }
	deleteBtn.OnTapped = func() { s.confirmDelete(task) }
}

// ── Actions ───────────────────────────────────────────────────────────────────

func (s *appState) refresh() {
	tasks, err := s.svc.All()
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	var filtered []*todo.Task
	for _, t := range tasks {
		switch s.filter {
		case "active":
			if !t.Done {
				filtered = append(filtered, t)
			}
		case "done":
			if t.Done {
				filtered = append(filtered, t)
			}
		default:
			filtered = append(filtered, t)
		}
	}
	s.tasks = filtered
	s.taskList.Refresh()

	total, done := len(tasks), 0
	for _, t := range tasks {
		if t.Done {
			done++
		}
	}
	s.statsLabel.SetText(fmt.Sprintf("%d / %d completed", done, total))
}

func (s *appState) toggleTask(t *todo.Task) {
	if err := s.svc.Toggle(t); err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.refresh()
}

func (s *appState) confirmDelete(t *todo.Task) {
	dialog.ShowConfirm("Delete Task",
		fmt.Sprintf("Delete \"%s\"?", t.Title),
		func(ok bool) {
			if ok {
				if err := s.svc.Delete(t.ID); err != nil {
					dialog.ShowError(err, s.win)
					return
				}
				s.refresh()
			}
		}, s.win)
}

func (s *appState) showTaskForm(existing *todo.Task) {
	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Task title…")

	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Optional description…")
	descEntry.SetMinRowsVisible(3)

	prioritySelect := widget.NewSelect([]string{"Low", "Medium", "High"}, nil)
	prioritySelect.SetSelected("Medium")

	if existing != nil {
		titleEntry.SetText(existing.Title)
		descEntry.SetText(existing.Description)
		prioritySelect.SetSelected(existing.Priority.String())
	}

	form := widget.NewForm(
		widget.NewFormItem("Title *", titleEntry),
		widget.NewFormItem("Description", descEntry),
		widget.NewFormItem("Priority", prioritySelect),
	)

	label := "Add Task"
	if existing != nil {
		label = "Edit Task"
	}

	dialog.ShowCustomConfirm(label, "Save", "Cancel", form, func(ok bool) {
		if !ok {
			return
		}
		if titleEntry.Text == "" {
			dialog.ShowError(fmt.Errorf("title cannot be empty"), s.win)
			return
		}
		pri := parsePriority(prioritySelect.Selected)
		var err error
		if existing == nil {
			_, err = s.svc.Add(titleEntry.Text, descEntry.Text, pri)
		} else {
			err = s.svc.Edit(existing, titleEntry.Text, descEntry.Text, pri)
		}
		if err != nil {
			dialog.ShowError(err, s.win)
			return
		}
		s.refresh()
	}, s.win)
}

func parsePriority(sel string) todo.Priority {
	switch sel {
	case "High":
		return todo.PriorityHigh
	case "Medium":
		return todo.PriorityMedium
	default:
		return todo.PriorityLow
	}
}

// ── Custom dark theme ─────────────────────────────────────────────────────────

type darkTheme struct{}

func (darkTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return colBackground
	case theme.ColorNameButton:
		return colAccent
	case theme.ColorNamePrimary:
		return colAccent
	case theme.ColorNameForeground:
		return color.White
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 35, G: 35, B: 50, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 80, G: 80, B: 100, A: 255}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 50, G: 50, B: 65, A: 255}
	}
	return theme.DefaultTheme().Color(n, v)
}

func (darkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (darkTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (darkTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNamePadding:
		return 10
	case theme.SizeNameText:
		return 14
	case theme.SizeNameInlineIcon:
		return 20
	}
	return theme.DefaultTheme().Size(n)
}
