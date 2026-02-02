package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"html"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/logic"
)

func (e *EditorWindow) createMenus() {
	mb := e.Window.MenuBar()

	// File
	fMenu := mb.AddMenu2("&File")

	// New Tab (Ctrl+N)
	actNew := fMenu.AddAction("&New File")
	actNew.SetShortcut(gui.NewQKeySequence2("Ctrl+N", gui.QKeySequence__NativeText))
	actNew.ConnectTriggered(func(bool) { e.TabManager.NewTab() })

	// Open File (Ctrl+O)
	actOpenFile := fMenu.AddAction("&Open File...")
	actOpenFile.SetShortcut(gui.NewQKeySequence2("Ctrl+O", gui.QKeySequence__NativeText))
	actOpenFile.ConnectTriggered(func(bool) {
		path := widgets.QFileDialog_GetOpenFileName(e.Window, "Open File", "", "All Files (*)", "", 0)
		if path != "" {
			e.OpenPath(path)
		}
	})

	// Open Project/Folder (Ctrl+Shift+O)
	actOpenFolder := fMenu.AddAction("Open &Folder/Project...")
	actOpenFolder.SetShortcut(gui.NewQKeySequence2("Ctrl+Shift+O", gui.QKeySequence__NativeText))
	actOpenFolder.ConnectTriggered(func(bool) {
		// Используем GetExistingDirectory для выбора именно папки
		path := widgets.QFileDialog_GetExistingDirectory(e.Window, "Open Project Folder", "", widgets.QFileDialog__ShowDirsOnly)
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

    eMenu.AddSeparator()

    eMenu.AddSeparator()

    actLineComplete := eMenu.AddAction("AI &Line Completion (Tab)")
    actLineComplete.SetCheckable(true)
    actLineComplete.SetChecked(e.TabManager.IsLineCompleteEnabled())
    actLineComplete.SetShortcut(gui.NewQKeySequence2("Ctrl+Shift+Space", gui.QKeySequence__NativeText))
    actLineComplete.ConnectTriggered(func(checked bool) {
        e.TabManager.SetLineCompleteEnabled(checked)
        if checked {
            e.Window.StatusBar().ShowMessage("AI Line Completion enabled (press Tab after code to trigger)", 3000)
        } else {
            e.Window.StatusBar().ShowMessage("AI Line Completion disabled", 2000)
        }
    })
    
    actAutoComplete := eMenu.AddAction("AI &Code Completion (Ctrl+L)")
    actAutoComplete.SetCheckable(true)
    actAutoComplete.SetChecked(e.TabManager.IsAutoCompleteEnabled())
    actAutoComplete.ConnectTriggered(func(checked bool) {
        e.TabManager.SetAutoCompleteEnabled(checked)
        if checked {
            e.Window.StatusBar().ShowMessage("AI Code Completion enabled (press Ctrl+L for multi-line completion)", 3000)
        } else {
            e.Window.StatusBar().ShowMessage("AI Code Completion disabled", 2000)
        }
    })
    

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
	
	// AI History Settings submenu
	aiHistoryMenu := vMenu.AddMenu2("AI &History Settings")
	
	// Пункт установки количества ответов для контекста
	actSetHistorySize := aiHistoryMenu.AddAction("Set History Context Size...")
	actSetHistorySize.ConnectTriggered(func(bool) {
		e.showHistoryContextSizeDialog()
	})
	
	// Пункт очистки истории
	actClearHistory := aiHistoryMenu.AddAction("Clear Conversation History")
	actClearHistory.ConnectTriggered(func(bool) {
		if len(e.AIResponseHistory) == 0 {
			e.Window.StatusBar().ShowMessage("History is already empty", 2000)
			return
		}
		
		btn := widgets.QMessageBox_Question(
			e.Window,
			"Clear History",
			fmt.Sprintf("Clear %d conversation entries from AI history?", len(e.AIResponseHistory)),
			widgets.QMessageBox__Yes|widgets.QMessageBox__No,
			widgets.QMessageBox__No,
		)
		
		if btn == widgets.QMessageBox__Yes {
			e.ClearAIHistory()
			e.UpdateAIContextDisplay()
		}
	})

	// Опция для включения контекста из всех открытых вкладок
	e.actUseTabsContext = aiHistoryMenu.AddAction("Use All Open Tabs as Context")
	e.actUseTabsContext.SetCheckable(true)
	e.actUseTabsContext.SetChecked(e.AIUseOpenTabsAsContext)
	e.actUseTabsContext.SetToolTip("Include the content of all open tabs as context for the AI")
	e.actUseTabsContext.ConnectTriggered(func(checked bool) {

		e.AIUseOpenTabsAsContext = checked
		e.UpdateAIContextDisplay() // Обновляем UI, чтобы показать изменение
		statusMsg := "Context from open tabs disabled"
		if checked {
			statusMsg = "Context from all open tabs is now enabled"
		}
		e.Window.StatusBar().ShowMessage(statusMsg, 3000)
	})

	
	// Показать текущий статус истории
	aiHistoryMenu.AddSeparator()
	actHistoryStatus := aiHistoryMenu.AddAction("Show History Status")
	actHistoryStatus.ConnectTriggered(func(bool) {
		msg := fmt.Sprintf("AI History Status:\n\n"+
			"• Total entries: %d\n"+
			"• Context size: %d\n"+
			"• Entries used for context: %d",
			len(e.AIResponseHistory),
			e.AIHistoryContextSize,
			min(e.AIHistoryContextSize, len(e.AIResponseHistory)))
		
		widgets.QMessageBox_Information(
			e.Window,
			"AI History Status",
			msg,
			widgets.QMessageBox__Ok,
			widgets.QMessageBox__Ok,
		)
	})

    vMenu.AddSeparator()

	// Подменю выбора стиля курсора
	cursorMenu := vMenu.AddMenu2("&Cursor Style")
	cursorGroup := widgets.NewQActionGroup(e.Window)
	cursorGroup.SetExclusive(true)

	for _, styleName := range CursorStyleOrder {
		style := CursorStyles[styleName]
		if style == nil {
			continue
		}
		
		currentStyleName := styleName // Захватываем для замыкания
		action := cursorMenu.AddAction(style.Name)
		action.SetCheckable(true)
		action.SetChecked(currentStyleName == e.TabManager.GetCurrentCursorStyleName())
		action.SetToolTip(style.Description)
		cursorGroup.AddAction(action)
		
		action.ConnectTriggered(func(checked bool) {
			if checked {
				e.TabManager.SetCursorStyle(currentStyleName)
			}
		})
	}

	vMenu.AddSeparator()

	//  Подменю выбора цветовой схемы
	schemeMenu := vMenu.AddMenu2("Color &Scheme")
	schemeGroup := widgets.NewQActionGroup(e.Window)
	schemeGroup.SetExclusive(true)

	// Сортируем схемы для стабильного порядка
	schemeNames := []string{"Monokai", "Dracula", "One Dark", "Solarized Dark", "GitHub Dark"}
	for _, name := range schemeNames {
		schemeName := name // Захватываем переменную для замыкания
		action := schemeMenu.AddAction(schemeName)
		action.SetCheckable(true)
		action.SetChecked(schemeName == e.TabManager.GetCurrentSchemeName())
		schemeGroup.AddAction(action)
		action.ConnectTriggered(func(checked bool) {
			if checked {
				e.TabManager.SetColorScheme(schemeName)
			}
		})
	}

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
	e.UpdateAIContextDisplay()
	
	// Формируем информацию о контексте для отображения пользователю
	var contextInfo []string

    // Защита от случайного редактирования
    e.AIChat.SetReadOnly(true)

	e.AIChat.Append(fmt.Sprintf("<b>You:</b> %s", prompt))

	contextStr := ""

	// 0. История предыдущих диалогов с AI
	historyContext := e.GetAIHistoryContext()
	if historyContext != "" {
		contextStr += historyContext
		usedCount := e.AIHistoryContextSize
		if usedCount > len(e.AIResponseHistory) {
			usedCount = len(e.AIResponseHistory)
		}
		contextInfo = append(contextInfo, fmt.Sprintf("[%d prev. responses]", usedCount))
	}

	// 1. Текущий файл
	ed := e.TabManager.CurrentEditor()
	if ed != nil {
		fileName := "Untitled"
		if ed.FilePath != "" {
			fileName = filepath.Base(ed.FilePath)
		}
		contextStr += fmt.Sprintf("\nUser is editing file: %s\nContent:\n%s\n", fileName, ed.TextEdit.ToPlainText())
		contextInfo = append(contextInfo, fileName)
	}

	// 1.5 Контекст из других открытых вкладок (если опция включена)
	if e.AIUseOpenTabsAsContext {
		tabsContext, tabNames := e.TabManager.GetAllOpenTabsContext(ed)
		if tabsContext != "" {
			contextStr += tabsContext
			// Добавляем информацию в UI
			if len(tabNames) <= 3 {
				contextInfo = append(contextInfo, fmt.Sprintf("[other tabs: %s]", strings.Join(tabNames, ", ")))
			} else {
				contextInfo = append(contextInfo, fmt.Sprintf("[other tabs: %s, ... +%d]", strings.Join(tabNames[:3], ", "), len(tabNames)-3))
			}

			// contextInfo = append(contextInfo, fmt.Sprintf("[%d other tabs]", len(tabNames)))
		}
	}


	// 2. Файлы проекта
	projectFiles := e.ProjectManager.GetContextFiles()
	if len(projectFiles) > 0 {
		ctx, _ := e.FileManager.CollectSpecificFilesContext(projectFiles)
		contextStr += ctx
		for _, f := range projectFiles {
			contextInfo = append(contextInfo, filepath.Base(f))
		}
	}

	//   3. Буфер обмена (если включён) ===
    if e.AIClipboardCheckbox != nil && e.AIClipboardCheckbox.IsChecked() {
		clipboard := gui.QGuiApplication_Clipboard()
		clipText := clipboard.Text(gui.QClipboard__Clipboard)

		if clipText != "" {
			// Ограничиваем размер буфера обмена (например, 10000 символов)
			if len(clipText) > 10000 {
				clipText = clipText[:10000] + "\n... [truncated]"
			}
			contextStr += fmt.Sprintf("\n--- Clipboard Content ---\n%s\n--- End Clipboard ---\n", clipText)
			contextInfo = append(contextInfo, "[clipboard]")
		}
	}

	// Показываем контекст в чате ===
	if len(contextInfo) > 0 {
		e.AIChat.Append(fmt.Sprintf("<span style='color:#888; font-size:10px;'>Context: %s</span>", 
			strings.Join(contextInfo, ", ")))
	}
	
	e.AIChat.Append("<i>Thinking...</i>")

	fullPrompt := contextStr + "\nUser Request: " + prompt

	go func() {

		resp, err := logic.SendMessageToLLM(fullPrompt, e.LLMProvider, e.LLMModel, e.LLMKey)
		e.RunOnUIThread(func() {
			if err != nil {
				e.AIChat.Append(fmt.Sprintf("<span style='color:red'>Error: %v</span>", err))
			} else {
				// NEW: Очищаем предыдущие кодовые блоки
				e.CurrentCodeBlocks = make([]CodeBlockData, 0)
				
				// Интеллектуальная обработка ответа для сохранения отступов в коде
				var finalHtml strings.Builder
				parts := strings.Split(resp, "```") // Разделяем ответ на текст и код

				codeBlockIndex := 0 // NEW: Счётчик блоков кода

				for i, part := range parts {
					// Пропускаем пустые части
					if strings.TrimSpace(part) == "" {
						continue
					}
					
					if i%2 == 0 {
						// Обычный текст
						escapedText := html.EscapeString(part)
						finalHtml.WriteString(fmt.Sprintf(
							`<div style="white-space: pre-wrap; word-wrap: break-word;">%s</div>`,
							escapedText,
						))
					} else {
						// Блок кода
						codeContent := part
						language := ""
						
						// Извлекаем язык программирования
						if nlIndex := strings.Index(part, "\n"); nlIndex != -1 {
							langHint := strings.TrimSpace(part[:nlIndex])
							if len(langHint) < 10 && !strings.Contains(langHint, " ") {
								language = langHint
								codeContent = part[nlIndex+1:]
							}
						}
						
						cleanCode := strings.TrimSpace(codeContent)
						escapedCode := html.EscapeString(cleanCode)

						// NEW: Сохраняем блок кода для последующего использования
						e.CurrentCodeBlocks = append(e.CurrentCodeBlocks, CodeBlockData{
							Code:     cleanCode,
							Language: language,
							Index:    codeBlockIndex,
						})

						// NEW: Добавляем кнопку над кодом
						langLabel := language
						if langLabel == "" {
							langLabel = "code"
						}
						
                        finalHtml.WriteString(fmt.Sprintf(
                            `<div style="margin: 5px 0;">
                                <a href="copycode:%d" style="background-color: #4A90E2; color: white; padding: 5px 12px; text-decoration: none; border-radius: 4px; font-size: 11px; display: inline-block; margin-bottom: 5px;">
                                    📋 Copy %s to Editor
                                </a>
                            </div>`,
                            codeBlockIndex,
                            langLabel,
                        ))
                                               
						// Оборачиваем код в <pre><code>
						finalHtml.WriteString(fmt.Sprintf(
							`<pre style="background-color: #2E2E2E; color: #DCDCDC; padding: 10px; border-radius: 5px; white-space: pre-wrap; word-wrap: break-word; margin-top: 0;"><code>%s</code></pre>`,
							escapedCode,
						))
						// Увеличиваем счётчик
						codeBlockIndex++
					}
				}

				e.AIChat.Append(fmt.Sprintf("<b>AI:</b><br>%s", finalHtml.String()))

				// Сохраняем в историю успешный ответ (оригинальный, не HTML)
				e.AddToAIHistory(prompt, resp)
				// Обновляем отображение контекста
				e.UpdateAIContextDisplay()
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

// showHistoryContextSizeDialog показывает диалог для установки размера контекста истории AI
func (e *EditorWindow) showHistoryContextSizeDialog() {
	dlg := widgets.NewQInputDialog(e.Window, core.Qt__Dialog)
	dlg.SetWindowTitle("AI History Context Size")
	dlg.SetLabelText("Number of previous AI responses to include as context:\n(0 = disabled, recommended: 3-5)")
	dlg.SetInputMode(widgets.QInputDialog__IntInput)
	dlg.SetIntRange(0, 20) // от 0 (отключено) до 20
	dlg.SetIntValue(e.AIHistoryContextSize)

	if dlg.Exec() == int(widgets.QDialog__Accepted) {
		newSize := dlg.IntValue()
		e.AIHistoryContextSize = newSize
		
		if newSize == 0 {
			e.Window.StatusBar().ShowMessage("AI history context disabled", 2000)
		} else {
			e.Window.StatusBar().ShowMessage(
				fmt.Sprintf("AI history context set to %d responses", newSize), 2000)
		}
		
		// Обновляем отображение контекста
		e.UpdateAIContextDisplay()
	}
}

// min возвращает минимум из двух чисел (хелпер для Go < 1.21)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleCodeBlockClick обрабатывает клик по кнопке копирования кода
func (e *EditorWindow) handleCodeBlockClick(url string) {
    // Парсим URL вида "copycode:0", "copycode://0", "copycode:1" и т.д.
    if !strings.HasPrefix(url, "copycode:") {
    	return
    }
    
    indexStr := strings.TrimPrefix(url, "copycode:")
    indexStr = strings.TrimPrefix(indexStr, "//") // поддержка copycode://N
    
	var blockIndex int
	if _, err := fmt.Sscanf(indexStr, "%d", &blockIndex); err != nil {
		e.Window.StatusBar().ShowMessage("Error parsing code block index", 2000)
		return
	}
	
	// Проверяем валидность индекса
	if blockIndex < 0 || blockIndex >= len(e.CurrentCodeBlocks) {
		e.Window.StatusBar().ShowMessage("Code block not found", 2000)
		return
	}
	
	codeBlock := e.CurrentCodeBlocks[blockIndex]
	
	// Получаем текущий редактор
	ed := e.TabManager.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		// Если нет открытых вкладок, создаём новую
		e.TabManager.NewTab()
		ed = e.TabManager.CurrentEditor()
		if ed == nil || ed.TextEdit == nil {
			e.Window.StatusBar().ShowMessage("Cannot access editor", 2000)
			return
		}
	}
	
	// ИСПРАВЛЕНИЕ: Сохраняем текущий фокус AI чата, чтобы не потерять содержимое
	aiChatHadFocus := e.AIChat.HasFocus()
	
	// Переключаем фокус на редактор ПЕРЕД манипуляциями с курсором
	ed.TextEdit.SetFocus2()
	
	// Вставляем код в позицию курсора редактора
	cursor := ed.TextEdit.TextCursor()
	
	// Если есть выделение, заменяем его
	if cursor.HasSelection() {
		cursor.RemoveSelectedText()
	}
	
	// ИСПРАВЛЕНИЕ: Используем beginEditBlock для атомарной операции
	cursor.BeginEditBlock()
	cursor.InsertText(codeBlock.Code)
	cursor.EndEditBlock()
	
	// Устанавливаем обновленный курсор
	ed.TextEdit.SetTextCursor(cursor)
	
	// ИСПРАВЛЕНИЕ: Возвращаем фокус AI чату, если он был активен
	if aiChatHadFocus {
		e.AIChat.SetFocus2()
	}

	// Показываем сообщение об успехе
	langInfo := codeBlock.Language
	if langInfo == "" {
		langInfo = "Code"
	}
	e.Window.StatusBar().ShowMessage(
		fmt.Sprintf("%s copied to editor (%d lines)", langInfo, strings.Count(codeBlock.Code, "\n")+1), 
		3000,
	)
}
