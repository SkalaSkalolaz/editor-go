package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/logic"
)

// CodeEditorTab wraps the text edit and its metadata
type CodeEditorTab struct {
	Widget                 *widgets.QWidget
	TextEdit               *widgets.QTextEdit
	LineNumbers            *widgets.QTextEdit
	Highlighter            *UniversalSyntaxHighlighter
	SearchWidget           *SearchWidget
	FilePath               string // Empty if new file
	IsModified             bool
	SuggestionText         string // Текст предложения от LLM
	SuggestionStartPos     int    // Позиция начала вставки предложения
	SuggestionEndPos       int    // Позиция конца предложения (для удаления при отклонении)
	HasSuggestion          bool   // Флаг наличия активного предложения
	IsWaitingLLM           bool   // Флаг ожидания ответа от LLM
	IsLineSuggestion       bool
	BracketHighlightActive bool
	BracketPos1            int
	BracketPos2            int
}

// TabManager handles the QTabWidget and editor instances
type TabManager struct {
	Tabs                *widgets.QTabWidget
	Editors             []*CodeEditorTab
	FileManager         *logic.FileManager
	Parent              *EditorWindow
	ShowLineNumbers     bool
	CurrentScheme       *ColorScheme
	AutoCompleteEnabled bool
	LineCompleteEnabled bool
	CurrentCursorStyle  *CursorStyle
}

// Карта для автоматического закрытия скобок
var autoPairMap = map[string]string{
	"(":  ")",
	"{":  "}",
	"[":  "]",
	"\"": "\"",
}

// bracketPairs определяет пары скобок для подсветки
var bracketPairs = map[rune]rune{
	'(': ')',
	')': '(',
	'{': '}',
	'}': '{',
	'[': ']',
	']': '[',
}

// openingBrackets — множество открывающих скобок
var openingBrackets = map[rune]bool{
	'(': true,
	'{': true,
	'[': true,
}

// isBracket проверяет, является ли символ скобкой
func isBracket(ch rune) bool {
	_, ok := bracketPairs[ch]
	return ok
}

// isOpeningBracket проверяет, является ли скобка открывающей
func isOpeningBracket(ch rune) bool {
	return openingBrackets[ch]
}

// handleAutoPairing обрабатывает автоматическое добавление закрывающих скобок/кавычек.
func (tm *TabManager) handleAutoPairing(editor *CodeEditorTab) {
	// Получаем текущий курсор
	cursor := editor.TextEdit.TextCursor()

	// Сохраняем исходную позицию
	originalPos := cursor.Position()
	if originalPos == 0 {
		return // Нечего проверять в самом начале документа
	}

	// Перемещаем курсор на один символ назад, чтобы "захватить" только что введенный символ
	cursor.MovePosition(gui.QTextCursor__PreviousCharacter, gui.QTextCursor__KeepAnchor, 1)
	char := cursor.SelectedText()

	// Проверяем, есть ли этот символ в нашей карте
	closingChar, ok := autoPairMap[char]
	if !ok {
		return // Это не тот символ, который нас интересует
	}

	// Блокируем сигналы, чтобы избежать рекурсивного вызова textChanged
	editor.TextEdit.BlockSignals(true)

	// Возвращаем курсор на исходную позицию (без выделения)
	cursor.SetPosition(originalPos, gui.QTextCursor__MoveAnchor)

	// Вставляем закрывающий символ
	cursor.InsertText(closingChar)

	// Перемещаем курсор обратно в центр между скобками
	cursor.SetPosition(originalPos, gui.QTextCursor__MoveAnchor)
	editor.TextEdit.SetTextCursor(cursor)

	// Разблокируем сигналы
	editor.TextEdit.BlockSignals(false)
}

func NewTabManager(parent *EditorWindow) *TabManager {
	tm := &TabManager{
		Tabs:               widgets.NewQTabWidget(nil),
		FileManager:        parent.FileManager,
		Parent:             parent,
		CurrentScheme:      ColorSchemes["Monokai"],
		CurrentCursorStyle: CursorStyles["Block"],
	}

	tm.Tabs.SetTabsClosable(true)
	tm.Tabs.SetMovable(true)

	// Close Tab Handler
	tm.Tabs.ConnectTabCloseRequested(func(index int) {
		tm.CloseTab(index)
	})

	return tm
}

// NewTab creates a generic "Untitled" tab
func (tm *TabManager) NewTab() {
	tm.addTab("", "")
}

// OpenFile opens a file in a new tab or switches to it if open
func (tm *TabManager) OpenFile(path string) {
	// Check if already open
	for i, ed := range tm.Editors {
		if ed.FilePath == path {
			tm.Tabs.SetCurrentIndex(i)
			return
		}
	}

	content, err := tm.FileManager.ReadFile(path)
	if err != nil {
		widgets.QMessageBox_Critical(tm.Parent.Window, "Error", err.Error(), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
		return
	}

	tm.addTab(path, content)
}

func (tm *TabManager) getLangKeyByPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".js", ".ts", ".json":
		return "javascript"
	case ".html", ".htm", ".xml":
		return "html"
	case ".go":
		return "go"
	default:
		if path == "" {
			return "go"
		}
		return ""
	}
}

func (tm *TabManager) addTab(path, content string) {
	editor := &CodeEditorTab{
		FilePath:    path,
		BracketPos1: -1,
		BracketPos2: -1,
	}

	// UI Construction
	editor.Widget = widgets.NewQWidget(nil, 0)
	mainLayout := widgets.NewQVBoxLayout()
	mainLayout.SetContentsMargins(0, 0, 0, 0)

	// NEW: Горизонтальный layout для номеров строк + редактора
	editorLayout := widgets.NewQHBoxLayout()
	editorLayout.SetContentsMargins(0, 0, 0, 0)
	editorLayout.SetSpacing(0)

	// NEW: Виджет нумерации строк
	editor.LineNumbers = widgets.NewQTextEdit(nil)
	editor.LineNumbers.SetReadOnly(true)
	editor.LineNumbers.SetVerticalScrollBarPolicy(core.Qt__ScrollBarAlwaysOff)
	editor.LineNumbers.SetHorizontalScrollBarPolicy(core.Qt__ScrollBarAlwaysOff)
	editor.LineNumbers.SetFixedWidth(50)
	editor.LineNumbers.SetStyleSheet(`
		QTextEdit {
			background-color: #1e1e1e;
			color: #858585;
			border: none;
			border-right: 1px solid #3c3c3c;
			padding-right: 5px;
		}
	`)
	editor.LineNumbers.SetAlignment(core.Qt__AlignRight)
	// Скрыто по умолчанию, если флаг выключен
	editor.LineNumbers.SetVisible(tm.ShowLineNumbers)

	editor.TextEdit = widgets.NewQTextEdit(nil)

	font := gui.NewQFont2("Monospace", 11, 1, false)
	editor.TextEdit.SetStyleSheet(`
		QTextEdit { 
			background-color: #2b2b2b; 
			color: #dcdcdc; 
			border: none;
			selection-background-color: #214283;
		}
	`)
	editor.TextEdit.SetCursorWidth(2)
	editor.TextEdit.SetFont(font)
	editor.LineNumbers.SetFont(font)

	editor.TextEdit.SetTabStopDistance(font.PointSizeF() * 4)
	// Применяем текущий стиль курсора
	if tm.CurrentCursorStyle != nil {
		editor.TextEdit.SetCursorWidth(tm.CurrentCursorStyle.Width)
	}

	//  Применяем цветовую схему к редактору
	tm.applySchemeToEditor(editor)

	// Вставляем вызов нового метода вместо старого switch
	langKey := tm.getLangKeyByPath(path)

	if langKey != "" {
		editor.Highlighter = NewUniversalHighlighter(editor.TextEdit.Document(), langKey, tm.CurrentScheme)
	}

	editor.TextEdit.ConnectKeyPressEvent(func(event *gui.QKeyEvent) {
		key := event.Key()

		// Если есть активное предложение — обрабатываем Enter/другие клавиши
		if editor.HasSuggestion {
			if tm.HandleKeyForSuggestion(editor, key) {
				// Enter был нажат, предложение принято — не передаём событие дальше
				return
			}
			// Предложение отклонено, но событие нужно обработать (ввести символ)
		}

		// Ctrl+C: если нет выделения — копируем всю текущую строку
		if key == int(core.Qt__Key_C) && (event.Modifiers()&core.Qt__ControlModifier) != 0 {
			cursor := editor.TextEdit.TextCursor()

			// Если есть выделение — оставляем стандартное поведение (копировать выделение)
			if cursor.HasSelection() {
				editor.TextEdit.Copy()
				return
			}

			// Нет выделения — копируем целиком строку, где стоит курсор
			block := cursor.Block()
			if block.IsValid() {
				lineText := block.Text()

				cb := gui.QGuiApplication_Clipboard()
				cb.SetText(lineText, gui.QClipboard__Clipboard)
				cb.SetText(lineText, gui.QClipboard__Selection) // optional for Linux
				return
			}

			editor.TextEdit.Copy()
			return
		}

		// Ctrl+X: если нет выделения — вырезаем всю текущую строку (и копируем в буфер)
		if key == int(core.Qt__Key_X) && (event.Modifiers()&core.Qt__ControlModifier) != 0 {
			cursor := editor.TextEdit.TextCursor()

			// Если есть выделение — стандартное поведение (вырезать выделение)
			if cursor.HasSelection() {
				editor.TextEdit.Cut()
				return
			}

			block := cursor.Block()
			if !block.IsValid() {
				editor.TextEdit.Cut()
				return
			}

			lineText := block.Text()

			// Копируем строку в буфер обмена
			cb := gui.QGuiApplication_Clipboard()
			cb.SetText(lineText, gui.QClipboard__Clipboard)
			cb.SetText(lineText, gui.QClipboard__Selection) // optional for Linux

			doc := editor.TextEdit.Document()
			startPos := block.Position()

			// Следующий блок — начало следующей строки
			nextBlock := block.Next()

			// Атомарно для Undo
			cursor.BeginEditBlock()
			cursor.SetPosition(startPos, gui.QTextCursor__MoveAnchor)

			if nextBlock.IsValid() {
				// Выделяем до начала следующего блока (это включает перевод строки)
				cursor.SetPosition(nextBlock.Position(), gui.QTextCursor__KeepAnchor)
				cursor.RemoveSelectedText()
			} else {
				// Последняя строка: удаляем только текст блока (без "несуществующего"         )
				cursor.SetPosition(startPos+block.Length()-1, gui.QTextCursor__KeepAnchor)
				cursor.RemoveSelectedText()
			}

			cursor.EndEditBlock()

			// Ставим курсор в начало удалённой строки (логично: на startPos)
			// Если удаляли последнюю строку, startPos может оказаться > длины документа — подправим.
			newPos := startPos
			docLen := doc.CharacterCount() - 1 // Qt считает +1 служебный символ в конце
			if docLen < 0 {
				docLen = 0
			}

			cursor.SetPosition(newPos, gui.QTextCursor__MoveAnchor)
			editor.TextEdit.SetTextCursor(cursor)

			return
		}

		// Обработка Tab для однострочного автодополнения (Line Complete)
		if key == int(core.Qt__Key_Tab) && tm.LineCompleteEnabled {
			// Проверяем, не Shift+Tab ли это (для unindent)
			if event.Modifiers()&core.Qt__ShiftModifier != 0 {
				// Shift+Tab — стандартный unindent
				tm.UnindentSelection()
				return
			}

			// Проверяем условия для автодополнения
			cursor := editor.TextEdit.TextCursor()

			// Если есть выделение — делаем indent, а не автодополнение
			if cursor.HasSelection() {
				tm.IndentSelection()
				return
			}

			// Проверяем, есть ли текст перед курсором в текущей строке
			currentBlock := cursor.Block()
			posInBlock := cursor.PositionInBlock()
			textBeforeCursor := ""
			if posInBlock > 0 {
				textBeforeCursor = currentBlock.Text()[:posInBlock]
			}

			// Если перед курсором есть непробельный текст — запускаем однострочное автодополнение
			if strings.TrimSpace(textBeforeCursor) != "" {
				tm.TriggerLineComplete() // <-- ИЗМЕНЕНО: было TriggerAutoComplete()
				return
			}

			// Иначе — вставляем обычный Tab
			cursor.InsertText("	")
			return
		}

		// Для всех остальных клавиш — стандартная обработка
		editor.TextEdit.KeyPressEventDefault(event)
	})

	//  Синхронизация прокрутки номеров строк с редактором
	if vBar := editor.TextEdit.VerticalScrollBar(); vBar != nil {
		vBar.ConnectValueChanged(func(value int) {
			if editor.LineNumbers != nil {
				if lnVBar := editor.LineNumbers.VerticalScrollBar(); lnVBar != nil {
					lnVBar.SetValue(value)
				}
			}
		})
	}

	// Connect Cursor Position Changed for line highlighting
	editor.TextEdit.ConnectCursorPositionChanged(func() {
		tm.highlightCurrentLine(editor)
		// Проверяем и подсвечиваем скобки при изменении позиции курсора
		// tm.checkAndHighlightBrackets(editor)
	})

	// Обработка двойного клика для выделения скобок
	editor.TextEdit.ConnectMouseDoubleClickEvent(func(event *gui.QMouseEvent) {
		// Выполняем стандартное поведение (выделение слова)
		editor.TextEdit.MouseDoubleClickEventDefault(event)

		// Вызываем подсветку скобок только по двойному клику
		tm.checkAndHighlightBrackets(editor)
	})

	// Initial highlight (подсветка при открытии файла)
	tm.highlightCurrentLine(editor)

	// Отслеживание изменений в документе
	editor.TextEdit.Document().ConnectContentsChanged(func() {
		// Игнорируем изменения, если это вставка/удаление предложения LLM
		if editor.HasSuggestion || editor.IsWaitingLLM {
			return
		}

		if !editor.IsModified {
			editor.IsModified = true
			// Обновляем заголовок вкладки, добавляя звёздочку
			idx := tm.Tabs.IndexOf(editor.Widget)
			if idx >= 0 {
				currentTitle := tm.Tabs.TabText(idx)
				if !strings.HasSuffix(currentTitle, "*") {
					tm.Tabs.SetTabText(idx, currentTitle+"*")
				}
			}
		}
	})

	//  Загружаем текст.
	editor.TextEdit.SetPlainText(content)

	//  Начальное обновление номеров строк после загрузки текста
	tm.updateLineNumbers(editor)

	//  Добавляем виджеты в горизонтальный layout
	editorLayout.AddWidget(editor.LineNumbers, 0, 0)
	editorLayout.AddWidget(editor.TextEdit, 1, 0) // stretch = 1 для редактора

	//  Создаём панель поиска для этой вкладки
	editor.SearchWidget = NewSearchWidget(tm)

	// Важно: панель поиска добавляем ПЕРЕД редактором
	mainLayout.AddWidget(editor.SearchWidget.Widget, 0, 0)
	mainLayout.AddLayout(editorLayout, 1)
	editor.Widget.SetLayout(mainLayout)

	// Add to Tabs
	title := "Untitled"
	if path != "" {
		title = filepath.Base(path)
	}

	idx := tm.Tabs.AddTab(editor.Widget, title)
	tm.Tabs.SetCurrentIndex(idx)
	tm.Tabs.SetTabToolTip(idx, path)

	tm.Editors = append(tm.Editors, editor)
}

// CurrentEditor returns the editor for the currently active tab
func (tm *TabManager) CurrentEditor() *CodeEditorTab {
	idx := tm.Tabs.CurrentIndex()
	if idx < 0 || idx >= len(tm.Editors) {
		// Fallback: try to find by widget logic just in case indices are out of sync
	}

	currentWidget := tm.Tabs.CurrentWidget()
	if currentWidget == nil {
		return nil
	}

	for _, ed := range tm.Editors {
		if ed.Widget.Pointer() == currentWidget.Pointer() {
			return ed
		}
	}
	return nil
}

// SaveCurrent saves the active tab
// SaveCurrent saves the currently active tab
func (tm *TabManager) SaveCurrent() bool {
	return tm.SaveTab(tm.CurrentEditor())
}

// SaveTab saves a specific editor instance
func (tm *TabManager) SaveTab(ed *CodeEditorTab) bool {
	if ed == nil {
		return false
	}

	// If file hasn't been saved yet (Untitled), redirect to SaveAs logic
	if ed.FilePath == "" {
		return tm.SaveTabAs(ed)
	}

	return tm.saveEditor(ed, ed.FilePath)
}

// SaveAs triggers Save As dialog for the current tab
func (tm *TabManager) SaveAs() bool {
	return tm.SaveTabAs(tm.CurrentEditor())
}

// SaveTabAs triggers Save As dialog for a specific editor
func (tm *TabManager) SaveTabAs(ed *CodeEditorTab) bool {
	if ed == nil {
		return false
	}

	filter := "Go Source (*.go);;All Files (*)"
	path := widgets.QFileDialog_GetSaveFileName(tm.Parent.Window, "Save As", "", filter, "", 0)
	if path == "" {
		return false // User cancelled
	}

	return tm.saveEditor(ed, path)
}

func (tm *TabManager) saveEditor(ed *CodeEditorTab, path string) bool {
	content := ed.TextEdit.ToPlainText()
	err := tm.FileManager.WriteFile(path, content)
	if err != nil {
		widgets.QMessageBox_Critical(tm.Parent.Window, "Error", err.Error(), widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
		return false
	}

	// Run gofmt if it's a go file
	if filepath.Ext(path) == ".go" {
		if err := tm.FileManager.RunGoFmt(path); err == nil {
			// Reload content if fmt changed it
			newContent, _ := tm.FileManager.ReadFile(path)
			// Preserve cursor? Simplified: just set text
			cursor := ed.TextEdit.TextCursor()
			pos := cursor.Position()
			ed.TextEdit.SetPlainText(newContent)

			// Try restore cursor
			newCursor := ed.TextEdit.TextCursor()
			docLength := len(ed.TextEdit.ToPlainText())
			if pos > docLength {
				pos = docLength
			}
			if pos < 0 {
				pos = 0
			}
			newCursor.SetPosition(pos, gui.QTextCursor__MoveAnchor)
			ed.TextEdit.SetTextCursor(newCursor)
		}
	}

	ed.FilePath = path
	ed.IsModified = false

	// Update Tab Title (убираем звёздочку если была)
	idx := tm.Tabs.IndexOf(ed.Widget)
	if idx >= 0 {
		title := filepath.Base(path)
		// Убираем звёздочку — файл сохранён
		tm.Tabs.SetTabText(idx, title)
	}

	tm.Tabs.SetTabToolTip(idx, path)

	tm.Parent.Window.StatusBar().ShowMessage("Saved: "+filepath.Base(path), 2000)
	return true
}

// CloseTab handles the closing logic
func (tm *TabManager) CloseTab(index int) bool {
	var ed *CodeEditorTab
	var eIdx int

	found := false
	for i, e := range tm.Editors {
		// Используем Pointer() для надежности и здесь
		if tm.Tabs.Widget(index).Pointer() == e.Widget.Pointer() {
			ed = e
			eIdx = i
			found = true
			break
		}
	}

	if !found {
		return true
	} // Вкладка уже закрыта или не найдена

	// 2. Handle Unsaved Changes
	if ed.IsModified {
		// Формируем имя файла для отображения
		fileName := tm.Tabs.TabText(index)
		// Убираем звёздочку из имени для диалога
		fileName = strings.TrimSuffix(fileName, "*")
		if fileName == "" || fileName == "Untitled" {
			fileName = "Untitled"
		}

		btn := widgets.QMessageBox_Question(
			tm.Parent.Window,
			"Unsaved Changes",
			fmt.Sprintf("Do you want to save changes to \"%s\"?\n\nYour changes will be lost if you don't save them.", fileName),
			widgets.QMessageBox__Save|widgets.QMessageBox__Discard|widgets.QMessageBox__Cancel,
			widgets.QMessageBox__Save,
		)

		switch btn {
		case widgets.QMessageBox__Cancel:
			return false // Отмена закрытия вкладки
		case widgets.QMessageBox__Save:
			if !tm.SaveTab(ed) {
				return false // Сохранение не удалось или отменено пользователем
			}
		case widgets.QMessageBox__Discard:
			// Продолжаем закрытие без сохранения
		}
	}

	// 3. Remove from UI and internal slice
	tm.Tabs.RemoveTab(index)
	if ed.Widget != nil {
		ed.Widget.DeleteLater()
	}

	if eIdx < len(tm.Editors) {
		tm.Editors = append(tm.Editors[:eIdx], tm.Editors[eIdx+1:]...)
	}

	return true
}

func (tm *TabManager) getTabIndex(ed *CodeEditorTab) int {
	return tm.Tabs.IndexOf(ed.Widget)
}

// UpdateFileAfterRename checks if the renamed file is open and updates its path/title.
func (tm *TabManager) UpdateFileAfterRename(oldPath, newPath string) {
	for i, ed := range tm.Editors {
		if ed.FilePath == oldPath {
			ed.FilePath = newPath
			tm.Tabs.SetTabText(i, filepath.Base(newPath))
			tm.Tabs.SetTabToolTip(i, newPath)
			// Re-detect language if extension changed
			if tm.Parent.ProjectManager.IsActive {
				tm.Parent.Window.SetWindowTitle(tm.Parent.ProjectManager.RootPath + " - Go Lite IDE")
			}
			return
		}
	}
}

// highlightCurrentLine — упрощённая версия, устанавливает курсор в центр видимости
// Полноценная подсветка строки в therecipe/qt требует кастомного виджета
func (tm *TabManager) highlightCurrentLine(editor *CodeEditorTab) {
	if editor == nil || editor.TextEdit == nil {
		return
	}

	// Обновляем строку состояния с номером текущей строки
	cursor := editor.TextEdit.TextCursor()
	block := cursor.Block()
	lineNum := block.BlockNumber() + 1
	col := cursor.PositionInBlock() + 1

	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("Line: %d, Column: %d", lineNum, col), 0)
}

func (tm *TabManager) updateLineNumbers(editor *CodeEditorTab) {
	if editor == nil || editor.LineNumbers == nil || editor.TextEdit == nil {
		return
	}

	doc := editor.TextEdit.Document()
	lineCount := doc.BlockCount()
	// Если текстовое поле пустое, в нем все равно есть одна строка (один блок)
	if lineCount == 0 {
		lineCount = 1
	}

	// Проверяем, изменилось ли количество строк с последнего обновления.
	// Если нет, то нет смысла перерисовывать номера.
	// Это простая оптимизация, чтобы избежать лишних SetPlainText.
	currentLineNumberText := editor.LineNumbers.ToPlainText()
	currentLinesInPanel := len(strings.Split(currentLineNumberText, "\n"))
	if currentLinesInPanel == lineCount {
		return
	}

	// Создаем строки с номерами. Это очень быстрая операция.
	var sb strings.Builder
	for i := 1; i <= lineCount; i++ {
		sb.WriteString(fmt.Sprintf("%d\n", i))
	}

	// Блокируем сигналы, чтобы избежать рекурсивных вызовов, и обновляем текст.
	editor.LineNumbers.BlockSignals(true)
	editor.LineNumbers.SetPlainText(sb.String())
	editor.LineNumbers.BlockSignals(false)

	// Синхронизация прокрутки уже настроена в `addTab`,
	// поэтому дополнительно здесь ее вызывать не нужно.
}

// NEW: ToggleLineNumbers переключает отображение номеров строк
func (tm *TabManager) ToggleLineNumbers() {
	tm.ShowLineNumbers = !tm.ShowLineNumbers

	// Обновляем все открытые редакторы
	for _, editor := range tm.Editors {
		if editor.LineNumbers != nil {
			editor.LineNumbers.SetVisible(tm.ShowLineNumbers)
			if tm.ShowLineNumbers {
				tm.updateLineNumbers(editor)
			}
		}
	}
}

// NEW: IsLineNumbersVisible возвращает текущее состояние отображения номеров
func (tm *TabManager) IsLineNumbersVisible() bool {
	return tm.ShowLineNumbers
}

// ShowSearch показывает панель поиска для текущего редактора
func (tm *TabManager) ShowSearch() {
	if ed := tm.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
		ed.SearchWidget.Show()
	}
}

// ShowSearchReplace показывает панель поиска с заменой
func (tm *TabManager) ShowSearchReplace() {
	if ed := tm.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
		ed.SearchWidget.ShowWithReplace()
	}
}

// HideSearch скрывает панель поиска
func (tm *TabManager) HideSearch() {
	if ed := tm.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
		ed.SearchWidget.Hide()
	}
}

// FindNext переходит к следующему результату поиска
func (tm *TabManager) FindNext() {
	if ed := tm.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
		if !ed.SearchWidget.IsVisible() {
			ed.SearchWidget.Show()
		} else {
			ed.SearchWidget.FindNext()
		}
	}
}

// FindPrev переходит к предыдущему результату поиска
func (tm *TabManager) FindPrev() {
	if ed := tm.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
		if !ed.SearchWidget.IsVisible() {
			ed.SearchWidget.Show()
		} else {
			ed.SearchWidget.FindPrev()
		}
	}
}

// GoToLine переходит к указанной строке и подсвечивает её
func (tm *TabManager) GoToLine(lineNumber int) {
	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	doc := ed.TextEdit.Document()
	totalLines := doc.BlockCount()

	// Проверка границ
	if lineNumber < 1 {
		lineNumber = 1
	}
	if lineNumber > totalLines {
		lineNumber = totalLines
	}

	// Находим блок (строку) по номеру (нумерация блоков с 0)
	block := doc.FindBlockByLineNumber(lineNumber - 1)
	if !block.IsValid() {
		return
	}

	// Перемещаем курсор в начало нужной строки
	cursor := ed.TextEdit.TextCursor()
	cursor.SetPosition(block.Position(), gui.QTextCursor__MoveAnchor)

	// Выделяем всю строку для визуального эффекта
	// therecipe/qt требует 3 аргумента: operation, mode, count
	cursor.MovePosition(gui.QTextCursor__EndOfBlock, gui.QTextCursor__KeepAnchor, 1)

	ed.TextEdit.SetTextCursor(cursor)

	// Прокручиваем к позиции (CenterCursor недоступен в QTextEdit, используем EnsureCursorVisible)
	ed.TextEdit.EnsureCursorVisible()

	// Фокус на редактор
	ed.TextEdit.SetFocus2()

	// Обновляем статус бар
	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("Jumped to line %d", lineNumber), 2000)
}

// ToggleComment комментирует/раскомментирует текущую строку или выделенные строки
func (tm *TabManager) ToggleComment() {
	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	cursor := ed.TextEdit.TextCursor()
	doc := ed.TextEdit.Document()

	// Определяем диапазон строк для обработки
	var startBlock, endBlock *gui.QTextBlock

	if cursor.HasSelection() {
		// Есть выделение — находим первый и последний блоки
		selStart := cursor.SelectionStart()
		selEnd := cursor.SelectionEnd()

		startBlock = doc.FindBlock(selStart)
		endBlock = doc.FindBlock(selEnd)

		// Если выделение заканчивается в начале строки, не включаем эту строку
		if selEnd == endBlock.Position() && startBlock.BlockNumber() != endBlock.BlockNumber() {
			endBlock = endBlock.Previous()
		}
	} else {
		// Нет выделения — работаем с текущей строкой
		startBlock = cursor.Block()
		endBlock = startBlock
	}

	if startBlock == nil || endBlock == nil || !startBlock.IsValid() || !endBlock.IsValid() {
		return
	}

	// Сохраняем номера блоков для последующего использования
	startBlockNum := startBlock.BlockNumber()
	endBlockNum := endBlock.BlockNumber()

	// Определяем операцию: комментировать или раскомментировать
	// Проверяем, все ли строки в диапазоне закомментированы
	allCommented := true
	block := startBlock
	for block != nil && block.IsValid() {
		text := block.Text()
		trimmed := strings.TrimLeft(text, " 	")
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			allCommented = false
			break
		}
		if block.BlockNumber() >= endBlockNum {
			break
		}
		block = block.Next()
	}

	// Начинаем групповое редактирование (для единого Undo)
	cursor.BeginEditBlock()

	// Обрабатываем каждую строку
	block = doc.FindBlockByNumber(startBlockNum)
	for block != nil && block.IsValid() {
		blockText := block.Text()

		// Позиционируем курсор в начало блока
		cursor.SetPosition(block.Position(), gui.QTextCursor__MoveAnchor)

		if allCommented {
			// Раскомментируем: удаляем "// " или "//"
			trimmed := strings.TrimLeft(blockText, " 	")
			if strings.HasPrefix(trimmed, "// ") {
				// Находим позицию "//" в строке
				idx := strings.Index(blockText, "//")
				if idx >= 0 {
					cursor.SetPosition(block.Position()+idx, gui.QTextCursor__MoveAnchor)
					// Удаляем "// " (3 символа)
					for i := 0; i < 3; i++ {
						cursor.DeleteChar()
					}
				}
			} else if strings.HasPrefix(trimmed, "//") {
				// Без пробела после //
				idx := strings.Index(blockText, "//")
				if idx >= 0 {
					cursor.SetPosition(block.Position()+idx, gui.QTextCursor__MoveAnchor)
					// Удаляем "//" (2 символа)
					for i := 0; i < 2; i++ {
						cursor.DeleteChar()
					}
				}
			}
		} else {
			// Комментируем: добавляем "// " в начало (после leading whitespace)
			if blockText != "" {
				// Находим первый непробельный символ
				leadingSpaces := len(blockText) - len(strings.TrimLeft(blockText, " 	"))
				cursor.SetPosition(block.Position()+leadingSpaces, gui.QTextCursor__MoveAnchor)
				cursor.InsertText("// ")
			}
		}

		if block.BlockNumber() >= endBlockNum {
			break
		}
		block = block.Next()
	}

	cursor.EndEditBlock()

	// Восстанавливаем выделение если было несколько строк
	if startBlockNum != endBlockNum {
		// Получаем обновлённые блоки после редактирования
		newStartBlock := doc.FindBlockByNumber(startBlockNum)
		newEndBlock := doc.FindBlockByNumber(endBlockNum)
		if newStartBlock != nil && newEndBlock != nil {
			cursor.SetPosition(newStartBlock.Position(), gui.QTextCursor__MoveAnchor)
			cursor.SetPosition(newEndBlock.Position()+newEndBlock.Length()-1, gui.QTextCursor__KeepAnchor)
			ed.TextEdit.SetTextCursor(cursor)
		}
	}

	// Статус
	action := "Commented"
	if allCommented {
		action = "Uncommented"
	}
	lineCount := endBlockNum - startBlockNum + 1
	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("%s %d line(s)", action, lineCount), 2000)
}

// IndentSelection добавляет отступ (табуляцию) к текущей строке или выделенным строкам
func (tm *TabManager) IndentSelection() {
	tm.shiftLines(true)
}

// UnindentSelection удаляет отступ (табуляцию) из текущей строки или выделенных строк
func (tm *TabManager) UnindentSelection() {
	tm.shiftLines(false)
}

// shiftLines — общая логика для сдвига строк влево/вправо
// indent=true — добавить отступ, indent=false — удалить отступ
func (tm *TabManager) shiftLines(indent bool) {
	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	cursor := ed.TextEdit.TextCursor()
	doc := ed.TextEdit.Document()

	// Определяем символ отступа (табуляция)
	// Можно заменить на пробелы: indentStr := "    " (4 пробела)
	const indentStr = "	"
	const indentLen = 1 // длина indentStr; если пробелы — поставьте 4

	// Определяем диапазон строк для обработки
	var startBlock, endBlock *gui.QTextBlock

	if cursor.HasSelection() {
		// Есть выделение — находим первый и последний блоки
		selStart := cursor.SelectionStart()
		selEnd := cursor.SelectionEnd()

		startBlock = doc.FindBlock(selStart)
		endBlock = doc.FindBlock(selEnd)

		// Если выделение заканчивается в начале строки, не включаем эту строку
		if selEnd == endBlock.Position() && startBlock.BlockNumber() != endBlock.BlockNumber() {
			endBlock = endBlock.Previous()
		}
	} else {
		// Нет выделения — работаем с текущей строкой
		startBlock = cursor.Block()
		endBlock = startBlock
	}

	if startBlock == nil || endBlock == nil || !startBlock.IsValid() || !endBlock.IsValid() {
		return
	}

	// Сохраняем номера блоков
	startBlockNum := startBlock.BlockNumber()
	endBlockNum := endBlock.BlockNumber()

	// Сохраняем позицию курсора для восстановления
	originalPos := cursor.Position()
	// originalAnchor := cursor.Anchor()
	hadSelection := cursor.HasSelection()

	// Начинаем групповое редактирование (для единого Undo)
	cursor.BeginEditBlock()

	linesModified := 0

	// Обрабатываем каждую строку
	block := doc.FindBlockByNumber(startBlockNum)
	for block != nil && block.IsValid() {
		blockText := block.Text()

		// Позиционируем курсор в начало блока
		cursor.SetPosition(block.Position(), gui.QTextCursor__MoveAnchor)

		if indent {
			// Добавляем отступ в начало строки
			cursor.InsertText(indentStr)
			linesModified++
		} else {
			// Удаляем отступ с начала строки
			if len(blockText) > 0 {
				firstChar := blockText[0]

				if firstChar == '	' {
					// Удаляем один символ табуляции
					cursor.DeleteChar()
					linesModified++
				} else if firstChar == ' ' {
					// Удаляем до 4 пробелов (или до первого непробельного символа)
					spacesToRemove := 0
					for i := 0; i < 4 && i < len(blockText) && blockText[i] == ' '; i++ {
						spacesToRemove++
					}
					if spacesToRemove > 0 {
						for i := 0; i < spacesToRemove; i++ {
							cursor.DeleteChar()
						}
						linesModified++
					}
				}
			}
		}

		if block.BlockNumber() >= endBlockNum {
			break
		}
		block = block.Next()
	}

	cursor.EndEditBlock()

	// Восстанавливаем выделение если было
	if hadSelection {
		// Пересчитываем позиции с учётом изменений
		newStartBlock := doc.FindBlockByNumber(startBlockNum)
		newEndBlock := doc.FindBlockByNumber(endBlockNum)

		if newStartBlock != nil && newEndBlock != nil && newStartBlock.IsValid() && newEndBlock.IsValid() {
			cursor.SetPosition(newStartBlock.Position(), gui.QTextCursor__MoveAnchor)
			// Выделяем до конца последней строки (без символа новой строки)
			endPos := newEndBlock.Position() + newEndBlock.Length() - 1
			if endPos < newEndBlock.Position() {
				endPos = newEndBlock.Position()
			}
			cursor.SetPosition(endPos, gui.QTextCursor__KeepAnchor)
			ed.TextEdit.SetTextCursor(cursor)
		}
	} else {
		// Восстанавливаем позицию курсора с учётом сдвига
		newPos := originalPos
		if indent {
			newPos += indentLen
		} else {
			// При удалении отступа позиция могла сдвинуться назад
			// Простое решение: оставляем курсор в начале строки
			currentBlock := doc.FindBlockByNumber(startBlockNum)
			if currentBlock != nil && currentBlock.IsValid() {
				newPos = currentBlock.Position()
			}
		}
		cursor.SetPosition(newPos, gui.QTextCursor__MoveAnchor)
		ed.TextEdit.SetTextCursor(cursor)
	}

	// Обновляем статус
	action := "Indented"
	if !indent {
		action = "Unindented"
	}
	lineCount := endBlockNum - startBlockNum + 1
	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("%s %d line(s)", action, lineCount), 2000)
}

// applySchemeToEditor применяет цветовую схему к редактору
func (tm *TabManager) applySchemeToEditor(editor *CodeEditorTab) {
	if editor == nil || editor.TextEdit == nil || tm.CurrentScheme == nil {
		return
	}

	scheme := tm.CurrentScheme

	// Применяем стили к текстовому редактору
	editor.TextEdit.SetStyleSheet(fmt.Sprintf(`
		QTextEdit { 
			background-color: %s; 
			color: %s; 
			border: none;
			selection-background-color: %s;
		}
	`, scheme.Background, scheme.Foreground, scheme.CurrentLine))

	// Применяем стили к номерам строк
	if editor.LineNumbers != nil {
		editor.LineNumbers.SetStyleSheet(fmt.Sprintf(`
			QTextEdit {
				background-color: %s;
				color: %s;
				border: none;
				border-right: 1px solid %s;
				padding-right: 5px;
			}
		`, scheme.Background, scheme.Comment, scheme.CurrentLine))
	}
}

// SetColorScheme устанавливает цветовую схему для всех редакторов
func (tm *TabManager) SetColorScheme(schemeName string) {
	scheme, ok := ColorSchemes[schemeName]
	if !ok {
		return
	}

	tm.CurrentScheme = scheme

	// Обновляем все открытые редакторы
	for _, editor := range tm.Editors {
		tm.applySchemeToEditor(editor)

		// Обновляем подсветчик если есть
		if editor.Highlighter != nil {
			editor.Highlighter.SetScheme(scheme)
		}
	}

	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("Color scheme changed to: %s", schemeName), 2000)
}

// GetCurrentSchemeName возвращает имя текущей схемы
func (tm *TabManager) GetCurrentSchemeName() string {
	if tm.CurrentScheme != nil {
		return tm.CurrentScheme.Name
	}
	return "Monokai"
}

func (tm *TabManager) EnableHighlighterForCurrentTab() {
	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	if ed.Highlighter != nil {
		return
	}

	// Используем универсальный подсветчик с определением языка по пути текущего файла
	langKey := tm.getLangKeyByPath(ed.FilePath)
	if langKey != "" {
		ed.Highlighter = NewUniversalHighlighter(ed.TextEdit.Document(), langKey, tm.CurrentScheme)
	}
}

// IsAutoCompleteEnabled возвращает состояние автодополнения
func (tm *TabManager) IsAutoCompleteEnabled() bool {
	return tm.AutoCompleteEnabled
}

// SetAutoCompleteEnabled устанавливает состояние автодополнения
func (tm *TabManager) SetAutoCompleteEnabled(enabled bool) {
	tm.AutoCompleteEnabled = enabled
}

// TriggerAutoComplete запускает процесс автодополнения для текущего редактора
func (tm *TabManager) TriggerAutoComplete() {
	if !tm.AutoCompleteEnabled {
		return
	}

	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	// Если уже ожидаем ответ от LLM — игнорируем
	if ed.IsWaitingLLM {
		return
	}

	// Если есть активное предложение — сначала его отклоняем
	if ed.HasSuggestion {
		tm.RejectSuggestion(ed)
	}

	// === НОВАЯ ЛОГИКА: Проверяем условия для генерации по комментарию ===
	if shouldGenerate, comment := tm.shouldTriggerCommentBasedGeneration(ed); shouldGenerate {
		tm.TriggerCommentBasedGeneration(comment)
		return
	}

	cursor := ed.TextEdit.TextCursor()
	doc := ed.TextEdit.Document()

	// Получаем текущую строку
	currentBlock := cursor.Block()
	currentLineText := currentBlock.Text()

	// Проверяем, что строка не пустая (есть начало кода)
	trimmedLine := strings.TrimSpace(currentLineText)
	if trimmedLine == "" {
		// Строка пустая и нет комментария выше — показываем подсказку
		tm.Parent.Window.StatusBar().ShowMessage("No code context. Add a comment above to generate code.", 3000)
		return
	}

	// Проверяем, что курсор не в начале строки (после слова)
	posInBlock := cursor.PositionInBlock()
	if posInBlock == 0 {
		// Курсор в начале строки — вставляем обычный Tab
		cursor.InsertText("	")
		return
	}

	// Проверяем, что перед курсором есть непробельный символ (слово)
	textBeforeCursor := currentLineText[:posInBlock]
	if strings.TrimSpace(textBeforeCursor) == "" {
		// Перед курсором только пробелы — вставляем обычный Tab
		cursor.InsertText("	")
		return
	}

	// === Собираем контекст: 10 строк до, текущая, 10 строк после ===
	currentLineNum := currentBlock.BlockNumber()
	totalLines := doc.BlockCount()

	startLine := currentLineNum - 10
	if startLine < 0 {
		startLine = 0
	}

	endLine := currentLineNum + 10
	if endLine >= totalLines {
		endLine = totalLines - 1
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("// Context lines before cursor:\n")

	// Собираем строки до текущей
	for i := startLine; i < currentLineNum; i++ {
		block := doc.FindBlockByNumber(i)
		if block.IsValid() {
			contextBuilder.WriteString(fmt.Sprintf("Line %d: %s\n", i+1, block.Text()))
		}
	}

	// Текущая строка с маркером позиции курсора
	contextBuilder.WriteString(fmt.Sprintf("\n// Current line (cursor at position %d):\n", posInBlock))
	contextBuilder.WriteString(fmt.Sprintf("Line %d: %s<CURSOR_HERE>\n", currentLineNum+1, textBeforeCursor))

	// Строки после текущей
	if currentLineNum < totalLines-1 {
		contextBuilder.WriteString("\n// Context lines after cursor:\n")
		for i := currentLineNum + 1; i <= endLine; i++ {
			block := doc.FindBlockByNumber(i)
			if block.IsValid() {
				contextBuilder.WriteString(fmt.Sprintf("Line %d: %s\n", i+1, block.Text()))
			}
		}
	}

	// Информация о файле
	fileInfo := "unknown file"
	if ed.FilePath != "" {
		fileInfo = ed.FilePath
	}

	// === Формируем промпт для LLM ===
	prompt := fmt.Sprintf(`You are a code completion assistant. Complete the code at the cursor position.

File: %s

%s

IMPORTANT INSTRUCTIONS:
1. Return ONLY the code that should be inserted at <CURSOR_HERE> position
2. Do NOT repeat any code that already exists before the cursor
3. Do NOT include any explanations - if explanation is needed, use code comments only
4. The completion should logically continue from what is already written
5. Keep the same coding style and indentation
6. Return raw code only, no markdown formatting, no backticks

Complete the code:`, fileInfo, contextBuilder.String())

	// Показываем индикатор загрузки
	ed.IsWaitingLLM = true
	ed.SuggestionStartPos = cursor.Position()
	tm.Parent.Window.StatusBar().ShowMessage("⏳ Waiting for AI suggestion...", 0)

	// Запускаем запрос к LLM в отдельной горутине
	go func() {
		resp, err := logic.SendMessageToLLM(prompt, tm.Parent.LLMProvider, tm.Parent.LLMModel, tm.Parent.LLMKey)

		tm.Parent.RunOnUIThread(func() {
			ed.IsWaitingLLM = false

			if err != nil {
				tm.Parent.Window.StatusBar().ShowMessage(fmt.Sprintf("AI Error: %v", err), 3000)
				return
			}

			// Очищаем ответ от возможных markdown-обёрток
			suggestion := tm.cleanLLMResponse(resp)

			if suggestion == "" {
				tm.Parent.Window.StatusBar().ShowMessage("AI returned empty suggestion", 2000)
				return
			}

			// Показываем предложение
			tm.ShowSuggestion(ed, suggestion)
		})
	}()
}

// ========== NEW: Методы для однострочного автодополнения ==========

// IsLineCompleteEnabled возвращает состояние однострочного автодополнения
func (tm *TabManager) IsLineCompleteEnabled() bool {
	return tm.LineCompleteEnabled
}

// SetLineCompleteEnabled устанавливает состояние однострочного автодополнения
func (tm *TabManager) SetLineCompleteEnabled(enabled bool) {
	tm.LineCompleteEnabled = enabled
}

// TriggerLineComplete запускает однострочное автодополнение для текущей строки
func (tm *TabManager) TriggerLineComplete() {
	if !tm.LineCompleteEnabled {
		tm.Parent.Window.StatusBar().ShowMessage("Line Completion is disabled. Enable it in Edit menu.", 3000)
		return
	}

	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	// Если уже ожидаем ответ от LLM — игнорируем
	if ed.IsWaitingLLM {
		tm.Parent.Window.StatusBar().ShowMessage("Already waiting for AI response...", 2000)
		return
	}

	// Если есть активное предложение — сначала его отклоняем
	if ed.HasSuggestion {
		tm.RejectSuggestion(ed)
	}

	cursor := ed.TextEdit.TextCursor()

	// Получаем текущую строку
	currentBlock := cursor.Block()
	currentLineText := currentBlock.Text()
	posInBlock := cursor.PositionInBlock()

	// Текст ДО курсора в текущей строке
	textBeforeCursor := ""
	if posInBlock > 0 {
		textBeforeCursor = currentLineText[:posInBlock]
	}

	// Текст ПОСЛЕ курсора в текущей строке
	textAfterCursor := ""
	if posInBlock < len(currentLineText) {
		textAfterCursor = currentLineText[posInBlock:]
	}

	// Проверяем, что перед курсором есть непробельный текст
	if strings.TrimSpace(textBeforeCursor) == "" {
		tm.Parent.Window.StatusBar().ShowMessage("No code before cursor to complete", 2000)
		return
	}

	// Собираем контекст: несколько строк до текущей для понимания
	doc := ed.TextEdit.Document()
	currentLineNum := currentBlock.BlockNumber()

	var contextBuilder strings.Builder

	// 5 строк до текущей для контекста
	startLine := currentLineNum - 5
	if startLine < 0 {
		startLine = 0
	}

	if startLine < currentLineNum {
		contextBuilder.WriteString("// Previous lines for context:\n")
		for i := startLine; i < currentLineNum; i++ {
			block := doc.FindBlockByNumber(i)
			if block.IsValid() {
				contextBuilder.WriteString(block.Text())
				contextBuilder.WriteString("\n")
			}
		}
		contextBuilder.WriteString("\n")
	}

	// Информация о файле
	fileInfo := "Go source file"
	if ed.FilePath != "" {
		fileInfo = ed.FilePath
	}

	// Формируем промпт специально для однострочного дополнения
	prompt := fmt.Sprintf(`You are a code completion assistant. Complete ONLY the current line.

File: %s

%s
// Current line to complete:
%s<CURSOR>%s

STRICT RULES:
1. Return ONLY the code that should be inserted at <CURSOR> position to complete THIS LINE
2. Do NOT add new lines - complete only the current line
3. Do NOT repeat any code that already exists before the cursor
4. Do NOT include explanations, markdown, or backticks
5. Keep the completion short and relevant to finish the statement/expression
6. If there is text after cursor, ensure your completion connects logically with it
7. Return raw code only

Complete this line:`, fileInfo, contextBuilder.String(), textBeforeCursor, textAfterCursor)

	// Показываем индикатор загрузки
	ed.IsWaitingLLM = true
	ed.IsLineSuggestion = true // Помечаем как однострочное
	ed.SuggestionStartPos = cursor.Position()
	tm.Parent.Window.StatusBar().ShowMessage("⏳ Completing line...", 0)

	// Запускаем запрос к LLM в отдельной горутине
	go func() {
		// Используем более короткий таймаут для inline completion
		resp, err := logic.SendMessageToLLMWithTimeout(
			prompt,
			tm.Parent.LLMProvider,
			tm.Parent.LLMModel,
			tm.Parent.LLMKey,
			30*time.Second, // Короткий таймаут
		)

		tm.Parent.RunOnUIThread(func() {
			ed.IsWaitingLLM = false

			if err != nil {
				ed.IsLineSuggestion = false
				tm.Parent.Window.StatusBar().ShowMessage(fmt.Sprintf("AI Error: %v", err), 3000)
				return
			}

			// Очищаем ответ
			suggestion := tm.cleanLineResponse(resp)

			if suggestion == "" {
				ed.IsLineSuggestion = false
				tm.Parent.Window.StatusBar().ShowMessage("AI returned empty suggestion", 2000)
				return
			}

			// Показываем предложение
			tm.ShowSuggestion(ed, suggestion)
		})
	}()
}

// cleanLineResponse очищает ответ LLM для однострочного дополнения
func (tm *TabManager) cleanLineResponse(response string) string {
	response = strings.TrimSpace(response)

	// Удаляем markdown code blocks
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var cleaned []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock {
				cleaned = append(cleaned, line)
			}
		}
		response = strings.Join(cleaned, "\n")
	}

	response = strings.TrimSpace(response)

	// Для однострочного дополнения берём только первую строку
	if idx := strings.Index(response, "\n"); idx != -1 {
		response = response[:idx]
	}

	return strings.TrimSpace(response)
}

// detectLanguageFromPath определяет язык программирования по расширению файла
func (tm *TabManager) detectLanguageFromPath(filePath string) string {
	if filePath == "" {
		return "Go" // По умолчанию Go
	}

	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":    "Go",
		".py":    "Python",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".jsx":   "JavaScript (React)",
		".tsx":   "TypeScript (React)",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".cc":    "C++",
		".h":     "C/C++ Header",
		".hpp":   "C++ Header",
		".rs":    "Rust",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
		".kt":    "Kotlin",
		".scala": "Scala",
		".cs":    "C#",
		".lua":   "Lua",
		".sh":    "Bash/Shell",
		".bash":  "Bash",
		".zsh":   "Zsh",
		".sql":   "SQL",
		".html":  "HTML",
		".css":   "CSS",
		".scss":  "SCSS",
		".sass":  "Sass",
		".json":  "JSON",
		".yaml":  "YAML",
		".yml":   "YAML",
		".xml":   "XML",
		".md":    "Markdown",
		".r":     "R",
		".dart":  "Dart",
		".ex":    "Elixir",
		".exs":   "Elixir",
		".erl":   "Erlang",
		".hs":    "Haskell",
		".ml":    "OCaml",
		".fs":    "F#",
		".clj":   "Clojure",
		".vim":   "Vim Script",
		".pl":    "Perl",
		".pm":    "Perl",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}

	return "Go" // Fallback
}

// findCommentAboveCursor ищет ближайший комментарий выше текущей позиции курсора
// Возвращает текст комментария (без префикса //) и номер строки, или пустую строку если не найден
func (tm *TabManager) findCommentAboveCursor(ed *CodeEditorTab) (string, int) {
	if ed == nil || ed.TextEdit == nil {
		return "", -1
	}

	cursor := ed.TextEdit.TextCursor()
	doc := ed.TextEdit.Document()
	currentBlockNum := cursor.Block().BlockNumber()

	// Идём вверх от текущей строки
	for lineNum := currentBlockNum - 1; lineNum >= 0; lineNum-- {
		block := doc.FindBlockByNumber(lineNum)
		if !block.IsValid() {
			continue
		}

		lineText := block.Text()
		trimmed := strings.TrimSpace(lineText)

		// Пропускаем пустые строки
		if trimmed == "" {
			continue
		}

		// Проверяем, является ли строка комментарием
		if strings.HasPrefix(trimmed, "//") {
			// Извлекаем текст комментария без префикса
			commentText := strings.TrimPrefix(trimmed, "//")
			commentText = strings.TrimSpace(commentText)
			return commentText, lineNum
		}

		// Если встретили непустую строку, которая не комментарий — прекращаем поиск
		// (комментарий должен быть непосредственно перед курсором)
		break
	}

	return "", -1
}

// shouldTriggerCommentBasedGeneration проверяет условия для генерации кода по комментарию:
// 1. Курсор в начале строки (позиция 0 в блоке)
// 2. Выше есть комментарий
func (tm *TabManager) shouldTriggerCommentBasedGeneration(ed *CodeEditorTab) (bool, string) {
	if ed == nil || ed.TextEdit == nil {
		return false, ""
	}

	cursor := ed.TextEdit.TextCursor()

	// Проверяем, что курсор в начале строки
	posInBlock := cursor.PositionInBlock()
	if posInBlock != 0 {
		return false, ""
	}

	// Проверяем, что текущая строка пустая или курсор в самом начале
	currentBlock := cursor.Block()
	currentLineText := strings.TrimSpace(currentBlock.Text())

	// Допускаем либо полностью пустую строку, либо строку с только пробелами
	if currentLineText != "" {
		return false, ""
	}

	// Ищем комментарий выше
	comment, _ := tm.findCommentAboveCursor(ed)
	if comment == "" {
		return false, ""
	}

	return true, comment
}

// TriggerCommentBasedGeneration запускает генерацию кода на основе комментария выше курсора
func (tm *TabManager) TriggerCommentBasedGeneration(comment string) {
	ed := tm.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	// Если уже ожидаем ответ от LLM — игнорируем
	if ed.IsWaitingLLM {
		tm.Parent.Window.StatusBar().ShowMessage("Already waiting for AI response...", 2000)
		return
	}

	// Если есть активное предложение — сначала его отклоняем
	if ed.HasSuggestion {
		tm.RejectSuggestion(ed)
	}

	cursor := ed.TextEdit.TextCursor()
	doc := ed.TextEdit.Document()

	// Определяем язык программирования
	language := tm.detectLanguageFromPath(ed.FilePath)

	// Собираем контекст: 15 строк до текущей позиции для понимания структуры
	currentLineNum := cursor.Block().BlockNumber()

	var contextBuilder strings.Builder

	startLine := currentLineNum - 15
	if startLine < 0 {
		startLine = 0
	}

	contextBuilder.WriteString("// Existing code context:\n")
	for i := startLine; i < currentLineNum; i++ {
		block := doc.FindBlockByNumber(i)
		if block.IsValid() {
			contextBuilder.WriteString(block.Text())
			contextBuilder.WriteString("\n")
		}
	}

	// Также собираем несколько строк после курсора для контекста
	totalLines := doc.BlockCount()
	endLine := currentLineNum + 10
	if endLine >= totalLines {
		endLine = totalLines - 1
	}

	var afterContext strings.Builder
	if currentLineNum < totalLines-1 {
		afterContext.WriteString("\n// Code after cursor position:\n")
		for i := currentLineNum + 1; i <= endLine; i++ {
			block := doc.FindBlockByNumber(i)
			if block.IsValid() {
				afterContext.WriteString(block.Text())
				afterContext.WriteString("\n")
			}
		}
	}

	// Информация о файле
	fileInfo := "untitled"
	if ed.FilePath != "" {
		fileInfo = ed.FilePath
	}

	// Формируем промпт для генерации кода по комментарию
	prompt := fmt.Sprintf(`You are a code generation assistant. Generate code based on the comment instruction.

Programming Language: %s
File: %s

%s
// Comment instruction (implement this):
// %s

<CURSOR - INSERT CODE HERE>
%s

STRICT RULES:
1. Generate ONLY the code that implements the comment instruction above
2. Write code in %s language following its best practices and idioms
3. Do NOT include the comment itself in your response - just the implementation
4. Do NOT include any explanations outside of code comments
5. Do NOT use markdown formatting or backticks
6. Match the existing code style (indentation, naming conventions)
7. Make the code complete and functional
8. If the comment describes a function, include proper error handling where appropriate
9. Return raw code only

Generate the implementation:`, language, fileInfo, contextBuilder.String(), comment, afterContext.String(), language)

	// Показываем индикатор загрузки
	ed.IsWaitingLLM = true
	ed.SuggestionStartPos = cursor.Position()
	tm.Parent.Window.StatusBar().ShowMessage(fmt.Sprintf("🤖 Generating %s code for: %s", language, comment), 0)

	// Запускаем запрос к LLM в отдельной горутине
	go func() {
		resp, err := logic.SendMessageToLLM(
			prompt,
			tm.Parent.LLMProvider,
			tm.Parent.LLMModel,
			tm.Parent.LLMKey,
		)

		tm.Parent.RunOnUIThread(func() {
			ed.IsWaitingLLM = false

			if err != nil {
				tm.Parent.Window.StatusBar().ShowMessage(fmt.Sprintf("AI Error: %v", err), 3000)
				return
			}

			// Очищаем ответ
			suggestion := tm.cleanLLMResponse(resp)

			if suggestion == "" {
				tm.Parent.Window.StatusBar().ShowMessage("AI returned empty code", 2000)
				return
			}

			// Показываем предложение
			tm.ShowSuggestion(ed, suggestion)
		})
	}()
}

// cleanLLMResponse очищает ответ LLM от markdown и лишних символов
func (tm *TabManager) cleanLLMResponse(response string) string {
	response = strings.TrimSpace(response)

	// Удаляем markdown code blocks
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		var cleaned []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if inBlock || !strings.HasPrefix(response, "```") {
				cleaned = append(cleaned, line)
			}
		}
		response = strings.Join(cleaned, "\n")
	}

	return strings.TrimSpace(response)
}

// ShowSuggestion отображает предложение от LLM серым курсивом
func (tm *TabManager) ShowSuggestion(ed *CodeEditorTab, suggestion string) {
	if ed == nil || ed.TextEdit == nil || suggestion == "" {
		return
	}

	cursor := ed.TextEdit.TextCursor()

	// Сохраняем позицию начала предложения
	ed.SuggestionStartPos = cursor.Position()
	ed.SuggestionText = suggestion
	ed.HasSuggestion = true

	// Блокируем сигналы чтобы вставка не триггерила textChanged
	ed.TextEdit.BlockSignals(true)

	// Создаём формат для "призрачного" текста (серый, курсив)
	suggestionFormat := gui.NewQTextCharFormat()
	suggestionFormat.SetForeground(gui.NewQBrush3(gui.NewQColor3(128, 128, 128, 180), core.Qt__SolidPattern))
	suggestionFormat.SetFontItalic(true)

	// Устанавливаем формат для курсора перед вставкой
	cursor.SetCharFormat(suggestionFormat)

	// Вставляем предложение (будет использован установленный формат)
	cursor.InsertText(suggestion)

	// Запоминаем конечную позицию
	ed.SuggestionEndPos = cursor.Position()

	// Возвращаем курсор в начало предложения
	cursor.SetPosition(ed.SuggestionStartPos, gui.QTextCursor__MoveAnchor)
	ed.TextEdit.SetTextCursor(cursor)

	ed.TextEdit.BlockSignals(false)

	tm.Parent.Window.StatusBar().ShowMessage("💡 Press Enter to accept, any other key to reject", 0)
}

// AcceptSuggestion принимает предложение — делает текст постоянным
// AcceptSuggestion принимает предложение — делает текст постоянным
func (tm *TabManager) AcceptSuggestion(ed *CodeEditorTab) {
	if ed == nil || !ed.HasSuggestion {
		return
	}

	// Перемещаем курсор в конец предложения
	cursor := ed.TextEdit.TextCursor()
	cursor.SetPosition(ed.SuggestionEndPos, gui.QTextCursor__MoveAnchor)

	// Выделяем весь текст предложения
	cursor.SetPosition(ed.SuggestionStartPos, gui.QTextCursor__KeepAnchor)

	// Получаем текст предложения
	suggestionText := ed.SuggestionText

	// Удаляем "призрачный" текст
	cursor.RemoveSelectedText()

	// Вставляем тот же текст, но с обычным форматированием
	cursor.InsertText(suggestionText)

	ed.TextEdit.SetTextCursor(cursor)

	// Формируем сообщение в зависимости от типа дополнения
	msg := "✓ Suggestion accepted"
	if ed.IsLineSuggestion {
		msg = "✓ Line completed"
	}

	// Сбрасываем состояние
	ed.HasSuggestion = false
	ed.SuggestionText = ""
	ed.SuggestionStartPos = 0
	ed.SuggestionEndPos = 0
	ed.IsLineSuggestion = false // NEW: сбрасываем флаг однострочного

	tm.Parent.Window.StatusBar().ShowMessage(msg, 2000)
}

// RejectSuggestion отклоняет предложение — удаляет "призрачный" текст
func (tm *TabManager) RejectSuggestion(ed *CodeEditorTab) {
	if ed == nil || !ed.HasSuggestion {
		return
	}

	ed.TextEdit.BlockSignals(true)

	// Удаляем "призрачный" текст
	cursor := ed.TextEdit.TextCursor()
	cursor.SetPosition(ed.SuggestionStartPos, gui.QTextCursor__MoveAnchor)
	cursor.SetPosition(ed.SuggestionEndPos, gui.QTextCursor__KeepAnchor)
	cursor.RemoveSelectedText()

	ed.TextEdit.SetTextCursor(cursor)

	ed.TextEdit.BlockSignals(false)

	// Сбрасываем состояние
	ed.HasSuggestion = false
	ed.SuggestionText = ""
	ed.SuggestionStartPos = 0
	ed.SuggestionEndPos = 0
	ed.IsLineSuggestion = false // NEW: сбрасываем флаг однострочного

	tm.Parent.Window.StatusBar().ShowMessage("Suggestion rejected", 1000)
}

// HandleKeyForSuggestion обрабатывает нажатие клавиши при активном предложении
// Возвращает true, если событие обработано (Enter для принятия)
func (tm *TabManager) HandleKeyForSuggestion(ed *CodeEditorTab, key int) bool {
	if ed == nil || !ed.HasSuggestion {
		return false
	}

	if key == int(core.Qt__Key_Return) || key == int(core.Qt__Key_Enter) {
		tm.AcceptSuggestion(ed)
		return true // Событие обработано, не передаём дальше
	}

	// Любая другая клавиша — отклоняем предложение
	tm.RejectSuggestion(ed)
	return false // Пусть событие обработается нормально (введётся символ)
}

// HasUnsavedChanges проверяет, есть ли несохранённые изменения в любой вкладке
func (tm *TabManager) HasUnsavedChanges() bool {
	for _, ed := range tm.Editors {
		if ed.IsModified {
			return true
		}
	}
	return false
}

// GetUnsavedEditors возвращает список редакторов с несохранёнными изменениями
func (tm *TabManager) GetUnsavedEditors() []*CodeEditorTab {
	var unsaved []*CodeEditorTab
	for _, ed := range tm.Editors {
		if ed.IsModified {
			unsaved = append(unsaved, ed)
		}
	}
	return unsaved
}

// CloseAllTabs пытается закрыть все вкладки, возвращает true если все закрыты
func (tm *TabManager) CloseAllTabs() bool {
	// Закрываем с конца, чтобы индексы не сбивались
	for tm.Tabs.Count() > 0 {
		if !tm.CloseTab(tm.Tabs.Count() - 1) {
			return false // Пользователь отменил закрытие
		}
	}
	return true
}

// PromptSaveAll показывает диалог для сохранения всех несохранённых файлов
// Возвращает: true — можно продолжать (все сохранены или отброшены), false — отмена
func (tm *TabManager) PromptSaveAll() bool {
	unsaved := tm.GetUnsavedEditors()
	if len(unsaved) == 0 {
		return true
	}

	// Формируем список файлов
	var fileList strings.Builder
	for _, ed := range unsaved {
		name := "Untitled"
		if ed.FilePath != "" {
			name = filepath.Base(ed.FilePath)
		}
		fileList.WriteString(fmt.Sprintf("  • %s\n", name))
	}

	msg := fmt.Sprintf("The following %d file(s) have unsaved changes:\n%s\nDo you want to save all changes?",
		len(unsaved), fileList.String())

	btn := widgets.QMessageBox_Question(
		tm.Parent.Window,
		"Save All Changes?",
		msg,
		widgets.QMessageBox__SaveAll|widgets.QMessageBox__Discard|widgets.QMessageBox__Cancel,
		widgets.QMessageBox__SaveAll,
	)

	switch btn {
	case widgets.QMessageBox__Cancel:
		return false // Отмена выхода
	case widgets.QMessageBox__Discard:
		// Сбрасываем флаги модификации, чтобы CloseTab не спрашивал повторно
		for _, ed := range unsaved {
			ed.IsModified = false
		}
		return true
	case widgets.QMessageBox__SaveAll:
		// Сохраняем все
		for _, ed := range unsaved {
			if !tm.SaveTab(ed) {
				// Если сохранение одного файла не удалось — спрашиваем что делать
				fileName := "Untitled"
				if ed.FilePath != "" {
					fileName = filepath.Base(ed.FilePath)
				}

				retryBtn := widgets.QMessageBox_Warning(
					tm.Parent.Window,
					"Save Failed",
					fmt.Sprintf("Failed to save \"%s\".\n\nDo you want to continue without saving this file?", fileName),
					widgets.QMessageBox__Yes|widgets.QMessageBox__No,
					widgets.QMessageBox__No,
				)

				if retryBtn == widgets.QMessageBox__No {
					return false // Отмена выхода
				}
				// Иначе продолжаем, помечаем как не модифицированный
				ed.IsModified = false
			}
		}
		return true
	}

	return true
}

// SetCursorStyle устанавливает стиль курсора для всех редакторов
func (tm *TabManager) SetCursorStyle(styleName string) {
	style, ok := CursorStyles[styleName]
	if !ok {
		return
	}

	tm.CurrentCursorStyle = style

	// Обновляем все открытые редакторы
	for _, editor := range tm.Editors {
		if editor.TextEdit != nil {
			editor.TextEdit.SetCursorWidth(style.Width)
		}
	}

	tm.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("Cursor style changed to: %s", styleName), 2000)
}

// GetCurrentCursorStyleName возвращает имя текущего стиля курсора
func (tm *TabManager) GetCurrentCursorStyleName() string {
	if tm.CurrentCursorStyle != nil {
		return tm.CurrentCursorStyle.Name
	}
	return "Line"
}

// GetCursorStyleDescription возвращает описание текущего стиля
func (tm *TabManager) GetCursorStyleDescription() string {
	if tm.CurrentCursorStyle != nil {
		return tm.CurrentCursorStyle.Description
	}
	return ""
}

// checkAndHighlightBrackets проверяет, стоит ли курсор на скобке, и подсвечивает пару
func (tm *TabManager) checkAndHighlightBrackets(editor *CodeEditorTab) {
	if editor == nil || editor.TextEdit == nil {
		return
	}

	// Если подсветка уже активна, не перезаписываем её
	// (она будет сброшена только по Escape)
	if editor.BracketHighlightActive {
		return
	}

	cursor := editor.TextEdit.TextCursor()
	pos := cursor.Position()
	text := editor.TextEdit.ToPlainText()

	if len(text) == 0 {
		return
	}

	// Конвертируем текст в руны для корректной работы с Unicode
	runes := []rune(text)

	// Проверяем символ под курсором и перед курсором
	var bracketPos int = -1
	var bracketChar rune

	// Сначала проверяем символ под курсором
	if pos < len(runes) {
		ch := runes[pos]
		if isBracket(ch) {
			bracketPos = pos
			bracketChar = ch
		}
	}

	// Если под курсором нет скобки, проверяем символ перед курсором
	if bracketPos == -1 && pos > 0 {
		ch := runes[pos-1]
		if isBracket(ch) {
			bracketPos = pos - 1
			bracketChar = ch
		}
	}

	// Если скобка не найдена — выходим
	if bracketPos == -1 {
		return
	}

	// Ищем парную скобку
	matchPos := tm.findMatchingBracket(runes, bracketPos, bracketChar)
	if matchPos == -1 {
		return
	}

	// Подсвечиваем обе скобки
	tm.highlightBracketPair(editor, bracketPos, matchPos)
	editor.BracketHighlightActive = true

	tm.Parent.Window.StatusBar().ShowMessage(
		"Bracket pair highlighted. Press Escape to clear.", 0)
}

// findMatchingBracket находит позицию парной скобки с учётом вложенности
func (tm *TabManager) findMatchingBracket(runes []rune, pos int, bracket rune) int {
	if pos < 0 || pos >= len(runes) {
		return -1
	}

	matchBracket := bracketPairs[bracket]
	isOpening := isOpeningBracket(bracket)

	// Направление поиска: вперёд для открывающих, назад для закрывающих
	direction := 1
	if !isOpening {
		direction = -1
	}

	// Счётчик вложенности
	depth := 1
	currentPos := pos + direction

	for currentPos >= 0 && currentPos < len(runes) {
		ch := runes[currentPos]

		if ch == bracket {
			// Нашли такую же скобку — увеличиваем глубину
			depth++
		} else if ch == matchBracket {
			// Нашли парную скобку — уменьшаем глубину
			depth--
			if depth == 0 {
				return currentPos
			}
		}

		currentPos += direction
	}

	return -1 // Парная скобка не найдена
}

// highlightBracketPair подсвечивает пару скобок через изменение фона символов
func (tm *TabManager) highlightBracketPair(editor *CodeEditorTab, pos1, pos2 int) {
	if editor == nil || editor.TextEdit == nil {
		return
	}

	// Сохраняем позиции для последующей очистки
	editor.BracketPos1 = pos1
	editor.BracketPos2 = pos2

	// Получаем цвет подсветки для текущей схемы
	highlightColor := GetBracketHighlightColor(tm.CurrentScheme)

	// Создаём формат для подсветки
	format := gui.NewQTextCharFormat()
	format.SetBackground(gui.NewQBrush3(highlightColor, core.Qt__SolidPattern))
	// Делаем текст жирным для лучшей видимости
	format.SetFontWeight(int(gui.QFont__Bold))

	// Блокируем сигналы чтобы изменения не триггерили события
	editor.TextEdit.BlockSignals(true)

	// Сохраняем текущую позицию курсора
	originalCursor := editor.TextEdit.TextCursor()
	originalPos := originalCursor.Position()

	// Подсвечиваем первую скобку
	cursor1 := editor.TextEdit.TextCursor()
	cursor1.SetPosition(pos1, gui.QTextCursor__MoveAnchor)
	cursor1.MovePosition(gui.QTextCursor__NextCharacter, gui.QTextCursor__KeepAnchor, 1)
	cursor1.MergeCharFormat(format)

	// Подсвечиваем вторую скобку
	cursor2 := editor.TextEdit.TextCursor()
	cursor2.SetPosition(pos2, gui.QTextCursor__MoveAnchor)
	cursor2.MovePosition(gui.QTextCursor__NextCharacter, gui.QTextCursor__KeepAnchor, 1)
	cursor2.MergeCharFormat(format)

	// Восстанавливаем позицию курсора
	originalCursor.SetPosition(originalPos, gui.QTextCursor__MoveAnchor)
	editor.TextEdit.SetTextCursor(originalCursor)

	editor.TextEdit.BlockSignals(false)
}

// ClearBracketHighlight очищает подсветку скобок для редактора
func (tm *TabManager) ClearBracketHighlight(editor *CodeEditorTab) {
	if editor == nil || editor.TextEdit == nil {
		return
	}

	if !editor.BracketHighlightActive {
		return
	}

	// ВАЖНО: Сначала сбрасываем флаг, чтобы избежать рекурсии
	editor.BracketHighlightActive = false

	// Блокируем сигналы, чтобы Rehighlight не вызвал CursorPositionChanged
	editor.TextEdit.BlockSignals(true)

	// Сбрасываем форматирование скобок напрямую
	tm.clearBracketFormat(editor, editor.BracketPos1)
	tm.clearBracketFormat(editor, editor.BracketPos2)

	// Сбрасываем сохранённые позиции
	editor.BracketPos1 = -1
	editor.BracketPos2 = -1

	editor.TextEdit.BlockSignals(false)

	tm.Parent.Window.StatusBar().ShowMessage("Bracket highlight cleared", 1500)
}

// clearBracketFormat очищает форматирование одной скобки
func (tm *TabManager) clearBracketFormat(editor *CodeEditorTab, pos int) {
	if pos < 0 {
		return
	}

	text := editor.TextEdit.ToPlainText()
	if pos >= len([]rune(text)) {
		return
	}

	// Сохраняем текущую позицию курсора
	originalCursor := editor.TextEdit.TextCursor()
	originalPos := originalCursor.Position()

	// Создаём формат по умолчанию
	defaultFormat := gui.NewQTextCharFormat()
	if tm.CurrentScheme != nil {
		defaultFormat.SetForeground(gui.NewQBrush3(
			hexToQColor(tm.CurrentScheme.Foreground),
			core.Qt__SolidPattern))
	}
	defaultFormat.SetFontWeight(int(gui.QFont__Normal))
	// Прозрачный фон
	defaultFormat.SetBackground(gui.NewQBrush2(core.Qt__NoBrush))

	// Применяем формат к позиции скобки
	cursor := editor.TextEdit.TextCursor()
	cursor.SetPosition(pos, gui.QTextCursor__MoveAnchor)
	cursor.MovePosition(gui.QTextCursor__NextCharacter, gui.QTextCursor__KeepAnchor, 1)
	cursor.SetCharFormat(defaultFormat)

	// Восстанавливаем позицию курсора
	originalCursor.SetPosition(originalPos, gui.QTextCursor__MoveAnchor)
	editor.TextEdit.SetTextCursor(originalCursor)
}

// ClearCurrentBracketHighlight очищает подсветку для текущего редактора
func (tm *TabManager) ClearCurrentBracketHighlight() {
	if ed := tm.CurrentEditor(); ed != nil {
		tm.ClearBracketHighlight(ed)
	}
}

// GetAllOpenTabsContext собирает содержимое всех открытых вкладок (кроме текущей)
func (tm *TabManager) GetAllOpenTabsContext(currentEditor *CodeEditorTab) (string, []string) {
	const (
		maxPerTabChars    = 120000  // лимит на 1 вкладку
		maxTotalTabsChars = 3000000 // общий лимит на все вкладки
	)

	var contextBuilder strings.Builder
	var fileNames []string

	totalAdded := 0
	hasHeader := false

	for _, ed := range tm.Editors {
		// Пропускаем текущий редактор, так как он уже добавляется отдельно
		if ed == nil || ed == currentEditor {
			continue
		}
		if ed.TextEdit == nil {
			continue
		}

		fileName := "Untitled"
		if ed.FilePath != "" {
			fileName = filepath.Base(ed.FilePath)
		}

		content := ed.TextEdit.ToPlainText()
		content = strings.TrimSpace(content)
		if content == "" {
			continue
		}

		// Ограничиваем контент на вкладку
		if len(content) > maxPerTabChars {
			content = content[:maxPerTabChars] + "\n... [truncated]"
		}

		// Проверяем общий лимит
		remaining := maxTotalTabsChars - totalAdded
		if remaining <= 0 {
			if !hasHeader {
				// даже если ничего не добавили, заголовок не нужен
			} else {
				contextBuilder.WriteString("\n... [other tabs context truncated: total limit reached]\n")
			}
			break
		}

		// Если текущий блок не помещается — тоже режем
		if len(content) > remaining {
			content = content[:remaining] + "\n... [truncated]"
		}

		if !hasHeader {
			contextBuilder.WriteString("\n--- Context from other open tabs ---\n")
			hasHeader = true
		}

		contextBuilder.WriteString(fmt.Sprintf("\nFile: %s\nContent:\n%s\n", fileName, content))
		fileNames = append(fileNames, fileName)

		totalAdded += len(content)
	}

	if hasHeader {
		contextBuilder.WriteString("\n--- End of other open tabs context ---\n")
	}

	return contextBuilder.String(), fileNames
}
