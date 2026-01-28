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

// CodeEditorTab wraps the text edit and its metadata
type CodeEditorTab struct {
	Widget      *widgets.QWidget
	TextEdit    *widgets.QTextEdit
    LineNumbers   *widgets.QTextEdit
    SearchWidget *SearchWidget
	Highlighter *SyntaxHighlighter
	FilePath    string // Empty if new file
	IsModified  bool
}

// TabManager handles the QTabWidget and editor instances
type TabManager struct {
	Tabs        *widgets.QTabWidget
	Editors     []*CodeEditorTab
	FileManager *logic.FileManager
	Parent      *EditorWindow
    ShowLineNumbers bool
}

func NewTabManager(parent *EditorWindow) *TabManager {
	tm := &TabManager{
		Tabs:        widgets.NewQTabWidget(nil),
		FileManager: parent.FileManager,
		Parent:      parent,
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

func (tm *TabManager) addTab(path, content string) {
	editor := &CodeEditorTab{
		FilePath: path,
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
	// Dark theme style for editor
	editor.TextEdit.SetStyleSheet(`
		QTextEdit { 
			background-color: #2b2b2b; 
			color: #dcdcdc; 
			border: none;
			selection-background-color: #214283;
		}
	`)
	// Включаем отображение курсора на всю ширину
	editor.TextEdit.SetCursorWidth(2)

	editor.TextEdit.SetFont(font)
	// NEW: Применяем тот же шрифт к номерам строк
	editor.LineNumbers.SetFont(font)
	
	// Tab stops (4 spaces)
	editor.TextEdit.SetTabStopDistance(font.PointSizeF() * 4) // Approx

	editor.TextEdit.SetPlainText(content)

	// Highlighter
	editor.Highlighter = NewSyntaxHighlighter(editor.TextEdit.Document())
	editor.Highlighter.SetLanguage(path)

	// Connect Modification Signal
	editor.TextEdit.ConnectTextChanged(func() {
		if !editor.IsModified {
			editor.IsModified = true
			idx := tm.getTabIndex(editor)
			if idx >= 0 {
				title := tm.Tabs.TabText(idx)
				if string(title[len(title)-1]) != "*" {
					tm.Tabs.SetTabText(idx, title+"*")
				}
			}
		}
		// NEW: Обновляем номера строк при изменении текста
		tm.updateLineNumbers(editor)
	})
	
	// NEW: Синхронизация прокрутки номеров строк с редактором
	editor.TextEdit.VerticalScrollBar().ConnectValueChanged(func(value int) {
		editor.LineNumbers.VerticalScrollBar().SetValue(value)
	})

    // Connect Cursor Position Changed for line highlighting
	editor.TextEdit.ConnectCursorPositionChanged(func() {
		tm.highlightCurrentLine(editor)
	})
	
	// Initial highlight (подсветка при открытии файла)
	tm.highlightCurrentLine(editor)
	
	//  Начальное обновление номеров строк
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
			newCursor.SetPosition(pos, gui.QTextCursor__MoveAnchor)
			ed.TextEdit.SetTextCursor(newCursor)
		}
	}

	ed.FilePath = path
	ed.IsModified = false
	ed.Highlighter.SetLanguage(path)
	
	// Update Tab Title
	idx := tm.Tabs.IndexOf(ed.Widget)
	tm.Tabs.SetTabText(idx, filepath.Base(path))
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

	if !found { return true } // Вкладка уже закрыта или не найдена

	// 2. Handle Unsaved Changes
	if ed.IsModified {
		btn := widgets.QMessageBox_Question(tm.Parent.Window, "Unsaved Changes", 
			fmt.Sprintf("Save changes to %s?", tm.Tabs.TabText(index)), 
			widgets.QMessageBox__Save|widgets.QMessageBox__Discard|widgets.QMessageBox__Cancel, 
			widgets.QMessageBox__Save)
		
		if btn == widgets.QMessageBox__Cancel { return false } // Abort closing
		if btn == widgets.QMessageBox__Save {
			if !tm.SaveTab(ed) { return false } // Save failed or cancelled
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

// updateLineNumbers обновляет содержимое виджета нумерации строк
// Учитывает визуальную высоту каждого блока при word wrap
func (tm *TabManager) updateLineNumbers(editor *CodeEditorTab) {
	if editor == nil || editor.LineNumbers == nil || editor.TextEdit == nil {
		return
	}
	
	doc := editor.TextEdit.Document()
	lineCount := doc.BlockCount()
	if lineCount < 1 {
		lineCount = 1
	}
	
	// ИСПРАВЛЕНИЕ: Получаем layout документа для вычисления высот блоков
	layout := doc.DocumentLayout()
	if layout == nil {
		// Fallback к простому варианту
		var lines []string
		for i := 1; i <= lineCount; i++ {
			lines = append(lines, fmt.Sprintf("%d", i))
		}
		editor.LineNumbers.BlockSignals(true)
		editor.LineNumbers.SetPlainText(strings.Join(lines, "\n"))
		editor.LineNumbers.BlockSignals(false)
		return
	}
	
	// Получаем высоту одной строки шрифта
	fontMetrics := gui.NewQFontMetrics(editor.TextEdit.Font())
	lineHeight := fontMetrics.LineSpacing()
	if lineHeight < 1 {
		lineHeight = 14 // fallback
	}
	
	var lines []string
	block := doc.Begin()
	lineNum := 1
	
	for block.IsValid() {
		// Получаем bounding rect блока
		blockRect := layout.BlockBoundingRect(block)
		blockHeight := int(blockRect.Height())
		
		// Вычисляем сколько визуальных строк занимает этот блок
		visualLines := blockHeight / lineHeight
		if visualLines < 1 {
			visualLines = 1
		}
		
		// Первая визуальная строка получает номер
		lines = append(lines, fmt.Sprintf("%d", lineNum))
		
		// Остальные визуальные строки (при wrap) получают пустые строки
		for i := 1; i < visualLines; i++ {
			lines = append(lines, "")
		}
		
		block = block.Next()
		lineNum++
	}
	
	editor.LineNumbers.BlockSignals(true)
	editor.LineNumbers.SetPlainText(strings.Join(lines, "\n"))
	editor.LineNumbers.BlockSignals(false)
	
	// Синхронизируем позицию прокрутки
	editor.LineNumbers.VerticalScrollBar().SetValue(
		editor.TextEdit.VerticalScrollBar().Value())
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
