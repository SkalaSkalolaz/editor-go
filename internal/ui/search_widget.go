package ui

import (
	"fmt"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

// SearchWidget — панель поиска и замены, встраиваемая в редактор
type SearchWidget struct {
	Widget        *widgets.QWidget
	SearchInput   *widgets.QLineEdit
	ReplaceInput  *widgets.QLineEdit
	BtnNext       *widgets.QPushButton
	BtnPrev       *widgets.QPushButton
	BtnReplace    *widgets.QPushButton
	BtnReplaceAll *widgets.QPushButton
	BtnClose      *widgets.QPushButton
	ChkCase       *widgets.QCheckBox
	ChkWholeWord  *widgets.QCheckBox
	LblStatus     *widgets.QLabel
	ReplaceRow    *widgets.QWidget

	TabManager    *TabManager
	CurrentMatches []TextMatch
	CurrentIndex   int
	searchHighlighter *SearchHighlighter
}

// TextMatch хранит позиции найденного текста
type TextMatch struct {
	Start int
	End   int
}

// SearchHighlighter добавляет подсветку найденных совпадений поверх основного highlighter
type SearchHighlighter struct {
	*gui.QSyntaxHighlighter
	SearchText   string
	CaseSensitive bool
	WholeWord    bool
	HighlightFormat *gui.QTextCharFormat
}

func NewSearchHighlighter(document *gui.QTextDocument) *SearchHighlighter {
	sh := &SearchHighlighter{
		QSyntaxHighlighter: gui.NewQSyntaxHighlighter2(document),
	}
	
	sh.HighlightFormat = gui.NewQTextCharFormat()
	sh.HighlightFormat.SetBackground(gui.NewQBrush3(
		gui.NewQColor3(255, 255, 0, 100), core.Qt__SolidPattern))
	
	sh.ConnectHighlightBlock(sh.highlightBlock)
	
	return sh
}

func (sh *SearchHighlighter) highlightBlock(text string) {
	if sh.SearchText == "" {
		return
	}
	
	searchText := sh.SearchText
	textToSearch := text
	
	if !sh.CaseSensitive {
		searchText = strings.ToLower(searchText)
		textToSearch = strings.ToLower(text)
	}
	
	startIndex := 0
	for {
		index := strings.Index(textToSearch[startIndex:], searchText)
		if index == -1 {
			break
		}
		
		actualIndex := startIndex + index
		
		// Проверка на целое слово
		if sh.WholeWord {
			if actualIndex > 0 && isWordChar(rune(text[actualIndex-1])) {
				startIndex = actualIndex + 1
				continue
			}
			endIndex := actualIndex + len(searchText)
			if endIndex < len(text) && isWordChar(rune(text[endIndex])) {
				startIndex = actualIndex + 1
				continue
			}
		}
		
		sh.SetFormat(actualIndex, len(searchText), sh.HighlightFormat)
		startIndex = actualIndex + len(searchText)
	}
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
	       (r >= '0' && r <= '9') || r == '_'
}

func (sh *SearchHighlighter) SetSearchParams(text string, caseSensitive, wholeWord bool) {
	sh.SearchText = text
	sh.CaseSensitive = caseSensitive
	sh.WholeWord = wholeWord
	sh.Rehighlight()
}

func (sh *SearchHighlighter) Clear() {
	sh.SearchText = ""
	sh.Rehighlight()
}

func NewSearchWidget(tabManager *TabManager) *SearchWidget {
	sw := &SearchWidget{
		TabManager:     tabManager,
		CurrentMatches: []TextMatch{},
		CurrentIndex:   -1,
	}

	sw.Widget = widgets.NewQWidget(nil, 0)
	sw.Widget.SetObjectName("SearchWidget")
	sw.Widget.SetStyleSheet(`
		#SearchWidget {
			background-color: #3c3c3c;
			border-bottom: 1px solid #555555;
			padding: 5px;
		}
		QLineEdit {
			background-color: #2b2b2b;
			color: #dcdcdc;
			border: 1px solid #555555;
			border-radius: 3px;
			padding: 3px 5px;
			min-width: 200px;
		}
		QLineEdit:focus {
			border-color: #007acc;
		}
		QPushButton {
			background-color: #4a4a4a;
			color: #dcdcdc;
			border: 1px solid #555555;
			border-radius: 3px;
			padding: 3px 10px;
			min-width: 60px;
		}
		QPushButton:hover {
			background-color: #5a5a5a;
		}
		QPushButton:pressed {
			background-color: #3a3a3a;
		}
		QCheckBox {
			color: #dcdcdc;
		}
		QLabel {
			color: #999999;
		}
	`)

	mainLayout := widgets.NewQVBoxLayout()
	mainLayout.SetContentsMargins(5, 5, 5, 5)
	mainLayout.SetSpacing(5)

	// === Первая строка: поиск ===
	searchRow := widgets.NewQHBoxLayout()
	searchRow.SetSpacing(5)

	lblSearch := widgets.NewQLabel2("Find:", nil, 0)
	lblSearch.SetStyleSheet("color: #dcdcdc; min-width: 50px;")
	sw.SearchInput = widgets.NewQLineEdit(nil)
	sw.SearchInput.SetPlaceholderText("Search...")

	sw.BtnPrev = widgets.NewQPushButton2("◀ Prev", nil)
	sw.BtnNext = widgets.NewQPushButton2("Next ▶", nil)
	sw.ChkCase = widgets.NewQCheckBox2("Case Sensitive", nil)
	sw.ChkWholeWord = widgets.NewQCheckBox2("Whole Word", nil)
	sw.LblStatus = widgets.NewQLabel2("", nil, 0)
	sw.LblStatus.SetMinimumWidth(80)

	sw.BtnClose = widgets.NewQPushButton2("✕", nil)
	sw.BtnClose.SetMaximumWidth(30)
	sw.BtnClose.SetStyleSheet("font-weight: bold;")

	searchRow.AddWidget(lblSearch, 0, 0)
	searchRow.AddWidget(sw.SearchInput, 1, 0)
	searchRow.AddWidget(sw.BtnPrev, 0, 0)
	searchRow.AddWidget(sw.BtnNext, 0, 0)
	searchRow.AddWidget(sw.ChkCase, 0, 0)
	searchRow.AddWidget(sw.ChkWholeWord, 0, 0)
	searchRow.AddWidget(sw.LblStatus, 0, 0)
	searchRow.AddStretch(0)
	searchRow.AddWidget(sw.BtnClose, 0, 0)

	mainLayout.AddLayout(searchRow, 0)

	// === Вторая строка: замена (скрыта по умолчанию) ===
	sw.ReplaceRow = widgets.NewQWidget(nil, 0)
	replaceLayout := widgets.NewQHBoxLayout()
	replaceLayout.SetContentsMargins(0, 0, 0, 0)
	replaceLayout.SetSpacing(5)

	lblReplace := widgets.NewQLabel2("Replace:", nil, 0)
	lblReplace.SetStyleSheet("color: #dcdcdc; min-width: 50px;")
	sw.ReplaceInput = widgets.NewQLineEdit(nil)
	sw.ReplaceInput.SetPlaceholderText("Replace with...")

	sw.BtnReplace = widgets.NewQPushButton2("Replace", nil)
	sw.BtnReplaceAll = widgets.NewQPushButton2("Replace All", nil)

	replaceLayout.AddWidget(lblReplace, 0, 0)
	replaceLayout.AddWidget(sw.ReplaceInput, 1, 0)
	replaceLayout.AddWidget(sw.BtnReplace, 0, 0)
	replaceLayout.AddWidget(sw.BtnReplaceAll, 0, 0)
	replaceLayout.AddStretch(1)

	sw.ReplaceRow.SetLayout(replaceLayout)
	sw.ReplaceRow.SetVisible(false)
	mainLayout.AddWidget(sw.ReplaceRow, 0, 0)

	sw.Widget.SetLayout(mainLayout)
	sw.Widget.SetVisible(false)

	// === Подключение сигналов ===
	sw.connectSignals()

	return sw
}

func (sw *SearchWidget) connectSignals() {
	// Поиск при вводе текста (с небольшой задержкой)
	sw.SearchInput.ConnectTextChanged(func(text string) {
		sw.performSearch()
	})

	// Enter в поле поиска — следующий результат
	sw.SearchInput.ConnectReturnPressed(func() {
		sw.FindNext()
	})

	// Enter в поле замены — заменить и перейти к следующему
	sw.ReplaceInput.ConnectReturnPressed(func() {
		sw.ReplaceCurrent()
	})

	// Кнопки
	sw.BtnNext.ConnectClicked(func(bool) { sw.FindNext() })
	sw.BtnPrev.ConnectClicked(func(bool) { sw.FindPrev() })
	sw.BtnReplace.ConnectClicked(func(bool) { sw.ReplaceCurrent() })
	sw.BtnReplaceAll.ConnectClicked(func(bool) { sw.ReplaceAll() })
	sw.BtnClose.ConnectClicked(func(bool) { sw.Hide() })

	// Чекбоксы — пересчитать поиск
	sw.ChkCase.ConnectStateChanged(func(int) { sw.performSearch() })
	sw.ChkWholeWord.ConnectStateChanged(func(int) { sw.performSearch() })
}

// Show показывает панель поиска (без замены)
func (sw *SearchWidget) Show() {
	sw.ReplaceRow.SetVisible(false)
	sw.Widget.SetVisible(true)
	sw.SearchInput.SetFocus2()
	sw.SearchInput.SelectAll()
	
	// Если есть выделенный текст — вставляем в поиск
	if ed := sw.TabManager.CurrentEditor(); ed != nil {
		selected := ed.TextEdit.TextCursor().SelectedText()
		if selected != "" && len(selected) < 100 {
			sw.SearchInput.SetText(selected)
			sw.SearchInput.SelectAll()
		}
	}
}

// ShowWithReplace показывает панель поиска с заменой
func (sw *SearchWidget) ShowWithReplace() {
	sw.ReplaceRow.SetVisible(true)
	sw.Widget.SetVisible(true)
	sw.SearchInput.SetFocus2()
	sw.SearchInput.SelectAll()
	
	if ed := sw.TabManager.CurrentEditor(); ed != nil {
		selected := ed.TextEdit.TextCursor().SelectedText()
		if selected != "" && len(selected) < 100 {
			sw.SearchInput.SetText(selected)
			sw.SearchInput.SelectAll()
		}
	}
}

// Hide скрывает панель и очищает подсветку
func (sw *SearchWidget) Hide() {
	sw.Widget.SetVisible(false)
	sw.clearHighlights()
	sw.CurrentMatches = []TextMatch{}
	sw.CurrentIndex = -1
	sw.LblStatus.SetText("")
	
	// Возвращаем фокус редактору
	if ed := sw.TabManager.CurrentEditor(); ed != nil {
		ed.TextEdit.SetFocus2()
	}
}

// Toggle переключает видимость панели поиска
func (sw *SearchWidget) Toggle() {
	if sw.Widget.IsVisible() {
		sw.Hide()
	} else {
		sw.Show()
	}
}

// performSearch выполняет поиск и подсвечивает все вхождения
func (sw *SearchWidget) performSearch() {
	sw.clearHighlights()
	sw.CurrentMatches = []TextMatch{}
	sw.CurrentIndex = -1

	searchText := sw.SearchInput.Text()
	if searchText == "" {
		sw.LblStatus.SetText("")
		return
	}

	ed := sw.TabManager.CurrentEditor()
	if ed == nil || ed.TextEdit == nil {
		return
	}

	document := ed.TextEdit.Document()
	content := ed.TextEdit.ToPlainText()

	// Настройка флагов поиска
	var flags gui.QTextDocument__FindFlag
	if sw.ChkCase.IsChecked() {
		flags |= gui.QTextDocument__FindCaseSensitively
	}
	if sw.ChkWholeWord.IsChecked() {
		flags |= gui.QTextDocument__FindWholeWords
	}

	// Ищем все вхождения
	cursor := gui.NewQTextCursor2(document)
	
	for {
		cursor = document.Find(searchText, cursor, flags)
		if cursor.IsNull() {
			break
		}
		
		sw.CurrentMatches = append(sw.CurrentMatches, TextMatch{
			Start: cursor.SelectionStart(),
			End:   cursor.SelectionEnd(),
		})
		
		// Защита от бесконечного цикла при пустом совпадении
		if cursor.SelectionStart() == cursor.SelectionEnd() {
			break
		}
	}

	// Подсвечиваем все вхождения
	sw.highlightAllMatches(ed)

	// Обновляем статус
	count := len(sw.CurrentMatches)
	if count == 0 {
		sw.LblStatus.SetText("No results")
		sw.LblStatus.SetStyleSheet("color: #ff6b6b;")
	} else {
		sw.LblStatus.SetText(fmt.Sprintf("%d found", count))
		sw.LblStatus.SetStyleSheet("color: #69db7c;")
		
		// Переходим к первому вхождению после курсора
		sw.findNextFromCursor(ed, content)
	}
}

// findNextFromCursor находит ближайшее вхождение после текущей позиции курсора
func (sw *SearchWidget) findNextFromCursor(ed *CodeEditorTab, content string) {
	if len(sw.CurrentMatches) == 0 {
		return
	}
	
	cursorPos := ed.TextEdit.TextCursor().Position()
	
	// Ищем первое вхождение после курсора
	for i, match := range sw.CurrentMatches {
		if match.Start >= cursorPos {
			sw.CurrentIndex = i
			sw.goToMatch(ed, i)
			return
		}
	}
	
	// Если не нашли — переходим к первому
	sw.CurrentIndex = 0
	sw.goToMatch(ed, 0)
}

// highlightAllMatches подсвечивает все найденные вхождения
func (sw *SearchWidget) highlightAllMatches(ed *CodeEditorTab) {
	if ed == nil || ed.TextEdit == nil {
		return
	}
	
	// Создаём или обновляем highlighter для подсветки поиска
	if sw.searchHighlighter == nil {
		sw.searchHighlighter = NewSearchHighlighter(ed.TextEdit.Document())
	}
	
	searchText := sw.SearchInput.Text()
	if searchText == "" || len(sw.CurrentMatches) == 0 {
		sw.searchHighlighter.Clear()
		return
	}
	
	sw.searchHighlighter.SetSearchParams(
		searchText,
		sw.ChkCase.IsChecked(),
		sw.ChkWholeWord.IsChecked(),
	)
}

// clearHighlights убирает подсветку
func (sw *SearchWidget) clearHighlights() {
	if sw.searchHighlighter != nil {
		sw.searchHighlighter.Clear()
	}
}

// goToMatch переходит к конкретному вхождению
func (sw *SearchWidget) goToMatch(ed *CodeEditorTab, index int) {
	if ed == nil || index < 0 || index >= len(sw.CurrentMatches) {
		return
	}

	match := sw.CurrentMatches[index]
	
	// Перемещаем курсор и выделяем текст
	cursor := ed.TextEdit.TextCursor()
	cursor.SetPosition(match.Start, gui.QTextCursor__MoveAnchor)
	cursor.SetPosition(match.End, gui.QTextCursor__KeepAnchor)
	ed.TextEdit.SetTextCursor(cursor)
	
	// Прокручиваем к позиции
	ed.TextEdit.EnsureCursorVisible()
	
	// Обновляем статус
	sw.LblStatus.SetText(fmt.Sprintf("%d of %d", index+1, len(sw.CurrentMatches)))
	sw.LblStatus.SetStyleSheet("color: #69db7c;")
}

// FindNext переходит к следующему вхождению
func (sw *SearchWidget) FindNext() {
	if len(sw.CurrentMatches) == 0 {
		sw.performSearch()
		return
	}

	ed := sw.TabManager.CurrentEditor()
	if ed == nil {
		return
	}

	sw.CurrentIndex++
	if sw.CurrentIndex >= len(sw.CurrentMatches) {
		sw.CurrentIndex = 0 // Циклический переход
	}

	sw.goToMatch(ed, sw.CurrentIndex)
}

// FindPrev переходит к предыдущему вхождению
func (sw *SearchWidget) FindPrev() {
	if len(sw.CurrentMatches) == 0 {
		sw.performSearch()
		return
	}

	ed := sw.TabManager.CurrentEditor()
	if ed == nil {
		return
	}

	sw.CurrentIndex--
	if sw.CurrentIndex < 0 {
		sw.CurrentIndex = len(sw.CurrentMatches) - 1 // Циклический переход
	}

	sw.goToMatch(ed, sw.CurrentIndex)
}

// ReplaceCurrent заменяет текущее выделенное вхождение
func (sw *SearchWidget) ReplaceCurrent() {
	if len(sw.CurrentMatches) == 0 || sw.CurrentIndex < 0 {
		return
	}

	ed := sw.TabManager.CurrentEditor()
	if ed == nil {
		return
	}

	replaceText := sw.ReplaceInput.Text()
	cursor := ed.TextEdit.TextCursor()

	// Проверяем, что выделен нужный текст
	if cursor.HasSelection() {
		cursor.InsertText(replaceText)
	}

	// Пересчитываем поиск и переходим к следующему
	sw.performSearch()
}

// ReplaceAll заменяет все вхождения
func (sw *SearchWidget) ReplaceAll() {
	searchText := sw.SearchInput.Text()
	if searchText == "" || len(sw.CurrentMatches) == 0 {
		return
	}

	ed := sw.TabManager.CurrentEditor()
	if ed == nil {
		return
	}

	replaceText := sw.ReplaceInput.Text()
	count := len(sw.CurrentMatches)

	// Блокируем обновление UI во время массовой замены
	ed.TextEdit.SetUpdatesEnabled(false)
	
	// Создаём единый блок отмены
	cursor := ed.TextEdit.TextCursor()
	cursor.BeginEditBlock()

	// Заменяем с конца, чтобы не сбивались позиции
	for i := len(sw.CurrentMatches) - 1; i >= 0; i-- {
		match := sw.CurrentMatches[i]
		cursor.SetPosition(match.Start, gui.QTextCursor__MoveAnchor)
		cursor.SetPosition(match.End, gui.QTextCursor__KeepAnchor)
		cursor.InsertText(replaceText)
	}

	cursor.EndEditBlock()
	ed.TextEdit.SetUpdatesEnabled(true)

	// Пересчитываем поиск
	sw.performSearch()

	// Показываем сообщение
	sw.TabManager.Parent.Window.StatusBar().ShowMessage(
		fmt.Sprintf("Replaced %d occurrences", count), 3000)
}

// IsVisible проверяет видимость панели
func (sw *SearchWidget) IsVisible() bool {
	return sw.Widget.IsVisible()
}

