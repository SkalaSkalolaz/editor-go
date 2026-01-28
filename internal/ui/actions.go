package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/logic"
)

func (e *EditorWindow) createMenus() {
	mb := e.Window.MenuBar()

	// File
	fMenu := mb.AddMenu2("&File")

	// New Tab (Ctrl+T)
	actNew := fMenu.AddAction("&New File")
	actNew.SetShortcut(gui.NewQKeySequence2("Ctrl+N", gui.QKeySequence__NativeText))
	actNew.ConnectTriggered(func(bool) { e.TabManager.NewTab() })

	// Open (Ctrl+O)
	actOpen := fMenu.AddAction("&Open File/Project...")
	actOpen.SetShortcut(gui.NewQKeySequence2("Ctrl+O", gui.QKeySequence__NativeText))
	actOpen.ConnectTriggered(func(bool) {
		path := widgets.QFileDialog_GetOpenFileName(e.Window, "Open", "", "All Files (*)", "", 0)
		if path != "" {
			e.OpenPath(path)
		}
	})

	fMenu.AddSeparator()

	// Save (Ctrl+S)
	actSave := fMenu.AddAction("&Save")
	actSave.SetShortcut(gui.NewQKeySequence2("Ctrl+S", gui.QKeySequence__NativeText))
	actSave.ConnectTriggered(func(bool) { e.TabManager.SaveCurrent() })

	// Save As (Ctrl+Shift+S)
	actSaveAs := fMenu.AddAction("Save &As...")
	actSaveAs.SetShortcut(gui.NewQKeySequence2("Ctrl+Shift+S", gui.QKeySequence__NativeText))
	actSaveAs.ConnectTriggered(func(bool) { e.TabManager.SaveAs() })

	fMenu.AddSeparator()

	// Quit (Ctrl+Q)
	actQuit := fMenu.AddAction("&Quit")
	actQuit.SetShortcut(gui.NewQKeySequence2("Ctrl+Q", gui.QKeySequence__NativeText))
	actQuit.ConnectTriggered(func(bool) { e.Window.Close() })

	// Edit
	eMenu := mb.AddMenu2("&Edit")

	// Helper to get current text edit safely
	withEditor := func(f func(ed *widgets.QTextEdit)) {
		if editor := e.TabManager.CurrentEditor(); editor != nil && editor.TextEdit != nil {
			f(editor.TextEdit)
		}
	}

	actUndo := eMenu.AddAction("&Cancel")
	actUndo.SetShortcut(gui.NewQKeySequence2("Ctrl+Z", gui.QKeySequence__NativeText))
	actUndo.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.Undo() }) })

	actRedo := eMenu.AddAction("&Return")
	actRedo.SetShortcut(gui.NewQKeySequence2("Ctrl+Shift+Z", gui.QKeySequence__NativeText))
	actRedo.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.Redo() }) })

	eMenu.AddSeparator()

	actCut := eMenu.AddAction("&Remove")
	actCut.SetShortcut(gui.NewQKeySequence2("Ctrl+X", gui.QKeySequence__NativeText))
	actCut.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.Cut() }) })

	actCopy := eMenu.AddAction("&Copy")
	actCopy.SetShortcut(gui.NewQKeySequence2("Ctrl+C", gui.QKeySequence__NativeText))
	actCopy.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.Copy() }) })

	actPaste := eMenu.AddAction("&Paste")
	actPaste.SetShortcut(gui.NewQKeySequence2("Ctrl+V", gui.QKeySequence__NativeText))
	actPaste.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.Paste() }) })

	eMenu.AddSeparator()

	actSelAll := eMenu.AddAction("Select &All")
	actSelAll.SetShortcut(gui.NewQKeySequence2("Ctrl+A", gui.QKeySequence__NativeText))
	actSelAll.ConnectTriggered(func(bool) { withEditor(func(t *widgets.QTextEdit) { t.SelectAll() }) })

	eMenu.AddSeparator()

    // Toggle Comment (Ctrl+/)
    actToggleComment := eMenu.AddAction("Toggle &Comment")
    actToggleComment.SetShortcut(gui.NewQKeySequence2("Ctrl+/", gui.QKeySequence__NativeText))
    actToggleComment.ConnectTriggered(func(bool) { e.TabManager.ToggleComment() })
    
    eMenu.AddSeparator()

    // Indent (Ctrl+])
    actIndent := eMenu.AddAction("&Indent")
    actIndent.SetShortcut(gui.NewQKeySequence2("Ctrl+]", gui.QKeySequence__NativeText))
    actIndent.ConnectTriggered(func(bool) { e.TabManager.IndentSelection() })

    // Unindent (Ctrl+[)
    actUnindent := eMenu.AddAction("&Unindent")
    actUnindent.SetShortcut(gui.NewQKeySequence2("Ctrl+[", gui.QKeySequence__NativeText))
    actUnindent.ConnectTriggered(func(bool) { e.TabManager.UnindentSelection() })

    eMenu.AddSeparator()

	//   Search & Replace 
	actFind := eMenu.AddAction("&Find...")
	actFind.SetShortcut(gui.NewQKeySequence2("Ctrl+F", gui.QKeySequence__NativeText))
	actFind.ConnectTriggered(func(bool) { e.TabManager.ShowSearch() })

	actReplace := eMenu.AddAction("Find && &Replace...")
	actReplace.SetShortcut(gui.NewQKeySequence2("Ctrl+H", gui.QKeySequence__NativeText))
	actReplace.ConnectTriggered(func(bool) { e.TabManager.ShowSearchReplace() })

	actFindNext := eMenu.AddAction("Find &Next")
	actFindNext.SetShortcut(gui.NewQKeySequence2("F3", gui.QKeySequence__NativeText))
	actFindNext.ConnectTriggered(func(bool) { e.TabManager.FindNext() })

	actFindPrev := eMenu.AddAction("Find Pre&vious")
	actFindPrev.SetShortcut(gui.NewQKeySequence2("Shift+F3", gui.QKeySequence__NativeText))
	actFindPrev.ConnectTriggered(func(bool) { e.TabManager.FindPrev() })

    eMenu.AddSeparator()
    
    // Go to Line (Ctrl+G)
    actGoToLine := eMenu.AddAction("&Go to Line...")
    actGoToLine.SetShortcut(gui.NewQKeySequence2("Ctrl+G", gui.QKeySequence__NativeText))
    actGoToLine.ConnectTriggered(func(bool) { e.showGoToLineDialog() })
    

	// View
	vMenu := mb.AddMenu2("&View")
	vMenu.AddAction("Toggle Project Tree").ConnectTriggered(func(bool) {
		e.ProjectTree.DockWidget.SetVisible(!e.ProjectTree.DockWidget.IsVisible())
	})
	vMenu.AddAction("Toggle AI Panel").ConnectTriggered(func(bool) {
		e.AIDock.SetVisible(!e.AIDock.IsVisible())
	})
	vMenu.AddAction("Toggle Output").ConnectTriggered(func(bool) {
		e.OutputDock.SetVisible(!e.OutputDock.IsVisible())
	})

    vMenu.AddSeparator()
	actLineNumbers := vMenu.AddAction("Show Line Numbers")
	actLineNumbers.SetCheckable(true)
	actLineNumbers.SetChecked(e.TabManager.IsLineNumbersVisible())
	actLineNumbers.ConnectTriggered(func(checked bool) {
		e.TabManager.ToggleLineNumbers()
		actLineNumbers.SetChecked(e.TabManager.IsLineNumbersVisible())
	})

	// Run
	rMenu := mb.AddMenu2("&Run")

	actArgs := rMenu.AddAction("Set Run Arguments...")
 
	actArgs.ConnectTriggered(func(bool) {
		// Используем создание экземпляра диалога вместо статической функции GetText,
		// чтобы избежать паники рефлексии (reflect zero Value) в биндингах Qt.
		dlg := widgets.NewQInputDialog(e.Window, core.Qt__Dialog)
		dlg.SetWindowTitle("Run Arguments")
		dlg.SetLabelText("Enter arguments (space separated):")
		dlg.SetTextValue(e.RunArgs)
		dlg.SetInputMode(widgets.QInputDialog__TextInput)
		
		// Exec блокирует поток до закрытия окна. Возвращает 1 (Accepted), если нажали OK.
		if dlg.Exec() == int(widgets.QDialog__Accepted) {
			e.RunArgs = dlg.TextValue()
			e.Window.StatusBar().ShowMessage(fmt.Sprintf("Args set: %s", e.RunArgs), 3000)
		}
	})


	actRun := rMenu.AddAction("Run Go Code")
	actRun.SetShortcut(gui.NewQKeySequence2("Ctrl+R", gui.QKeySequence__NativeText))
	actRun.ConnectTriggered(func(bool) { e.runGoCode() })
}

func (e *EditorWindow) runGoCode() {
	if !e.TabManager.SaveCurrent() {
		return
	}

	targetDir := ""
	targetArgs := []string{"run"}

	ed := e.TabManager.CurrentEditor()
	if ed == nil {
		return
	}

    if e.ProjectManager.IsActive && e.ProjectManager.IsFileInProject(ed.FilePath) {
		targetDir = e.ProjectManager.RootPath
		
		// Вычисляем путь к пакету текущего файла относительно корня проекта
		dirOfFile := filepath.Dir(ed.FilePath)
		if relPath, err := filepath.Rel(targetDir, dirOfFile); err == nil {
			// Если мы в корне, оставляем ".", иначе формируем путь "./cmd/app"
			if relPath == "." {
				targetArgs = append(targetArgs, ".")
			} else {
				// Используем Separator для кроссплатформенности, добавляем префикс "./"
				targetArgs = append(targetArgs, "."+string(filepath.Separator)+relPath)
			}
		} else {
			// Fallback, если не удалось вычислить относительный путь
			targetArgs = append(targetArgs, ".")
		}
	} else {
		// Режим одиночного файла (вне проекта)
		targetDir = filepath.Dir(ed.FilePath)
		targetArgs = append(targetArgs, filepath.Base(ed.FilePath))
	}

	if e.RunArgs != "" {
		userArgs := strings.Fields(e.RunArgs)
		targetArgs = append(targetArgs, userArgs...)
	}

	e.OutputDock.Show()
	e.OutputText.AppendPlainText(fmt.Sprintf("\n--- Starting: go %v ---\n", targetArgs[1:]))

	e.BtnStop.SetEnabled(true)

	// Callback для вывода текста в UI (потокобезопасно)
	onOutput := func(text string) {
		e.RunOnUIThread(func() {
			// Перемещаем курсор в конец перед вставкой, чтобы эффект был как в терминале
			e.OutputText.MoveCursor(gui.QTextCursor__End, gui.QTextCursor__MoveAnchor)
			e.OutputText.InsertPlainText(text)
			sb := e.OutputText.VerticalScrollBar()
			sb.SetValue(sb.Maximum())
		})
	}

	go func() {
		doneChan, cancel := e.ProcessRunner.StartCommand(targetDir, "go", targetArgs, onOutput)

		e.RunOnUIThread(func() {
			// Важно: Отключаем старый обработчик перед подключением нового
			e.BtnStop.DisconnectClicked()
			e.BtnStop.ConnectClicked(func(bool) {
				cancel()
				e.OutputText.AppendPlainText("\n[Stopped by User]\n")
				e.BtnStop.SetEnabled(false)
			})
		})

		err := <-doneChan

		resultMsg := "Finished Successfully."
		if err != nil {
			resultMsg = fmt.Sprintf("Finished with Error: %v", err)
		}

		e.RunOnUIThread(func() {
			e.OutputText.AppendPlainText(fmt.Sprintf("\n>>> %s\n", resultMsg))
			e.BtnStop.SetEnabled(false)
			e.BtnStop.DisconnectClicked()
		})
	}()
}

func (e *EditorWindow) HandleAskLLM(prompt string) {
	e.AIDock.Show()
	e.AIChat.Append(fmt.Sprintf("<b>You:</b> %s", prompt))
	e.AIChat.Append("<i>Thinking...</i>")

	contextStr := ""

	ed := e.TabManager.CurrentEditor()
	if ed != nil {
		contextStr += fmt.Sprintf("\nUser is editing file: %s\nContent:\n%s\n", filepath.Base(ed.FilePath), ed.TextEdit.ToPlainText())
	}

	projectFiles := e.ProjectManager.GetContextFiles()
	if len(projectFiles) > 0 {
		ctx, _ := e.FileManager.CollectSpecificFilesContext(projectFiles)
		contextStr += ctx
	}

	fullPrompt := contextStr + "\nUser Request: " + prompt

	go func() {
		resp, err := logic.SendMessageToLLM(fullPrompt, e.LLMProvider, e.LLMModel, e.LLMKey)

		e.RunOnUIThread(func() {
			if err != nil {
				e.AIChat.Append(fmt.Sprintf("<span style='color:red'>Error: %v</span>", err))
			} else {
				html := strings.ReplaceAll(resp, "\n", "<br>")
				html = strings.ReplaceAll(html, "```", "<hr>")
				e.AIChat.Append(fmt.Sprintf("<b>AI:</b><br>%s<br><hr>", html))
			}
			sb := e.AIChat.VerticalScrollBar()
			sb.SetValue(sb.Maximum())
		})
	}()
}

func (e *EditorWindow) showGoToLineDialog() {
	ed := e.TabManager.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	// Получаем общее количество строк
	doc := ed.TextEdit.Document()
	totalLines := doc.BlockCount()
	if totalLines < 1 {
		totalLines = 1
	}

	// Получаем текущую строку для подсказки
	currentLine := ed.TextEdit.TextCursor().Block().BlockNumber() + 1

	// Создаём диалог ввода
	dlg := widgets.NewQInputDialog(e.Window, core.Qt__Dialog)
	dlg.SetWindowTitle("Go to Line")
	dlg.SetLabelText(fmt.Sprintf("Enter line number (1-%d):", totalLines))
	dlg.SetInputMode(widgets.QInputDialog__IntInput)
	dlg.SetIntRange(1, totalLines)
	dlg.SetIntValue(currentLine)

	if dlg.Exec() == int(widgets.QDialog__Accepted) {
		targetLine := dlg.IntValue()
		e.TabManager.GoToLine(targetLine)
	}
}
