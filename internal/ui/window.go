package ui

import (
	"os"
	"fmt"
    "path/filepath"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
    "github.com/therecipe/qt/gui"

	"go-gnome-editor/internal/logic"
)

// AIHistoryEntry представляет одну запись в истории диалога с AI
type AIHistoryEntry struct {
	UserPrompt   string // Запрос пользователя
	AIResponse   string // Ответ LLM
	Timestamp    string // Время запроса (для отображения)
}

type EditorWindow struct {
	Window         *widgets.QMainWindow
	TabManager     *TabManager
	FileManager    *logic.FileManager
	ProjectManager *logic.ProjectManager
	ProjectTree    *ProjectTreeWidget
	ProcessRunner  *logic.ProcessRunner

	// Panels
	OutputDock  *widgets.QDockWidget
	OutputText  *widgets.QPlainTextEdit
	AIDock      *widgets.QDockWidget
	AIChat      *widgets.QTextBrowser
	AIInput     *widgets.QPlainTextEdit

	// Controls
	BtnStop     *widgets.QPushButton

    // AI Panel Controls (NEW)
	AIClipboardCheckbox *widgets.QCheckBox
	AIContextLabel      *widgets.QLabel

    // Хранение кодовых блоков из AI ответов
    CurrentCodeBlocks   []CodeBlockData

    // Runtime Configuration
	RunArgs string

	// Logic State
	LLMProvider string
	LLMModel    string
	LLMKey      string

    // AI Chat History for context
    AIResponseHistory    []AIHistoryEntry
    AIHistoryContextSize int   
	AIUseOpenTabsAsContext bool

    actUseTabsContext *widgets.QAction
}

// CodeBlockData хранит информацию о блоке кода в AI чате
type CodeBlockData struct {
	Code     string
	Language string
	Index    int
}

func NewEditorWindow(provider, model, key string) *EditorWindow {
	ew := &EditorWindow{
		Window:         widgets.NewQMainWindow(nil, 0),
		FileManager:    logic.NewFileManager(),
		ProjectManager: logic.NewProjectManager(),
		ProcessRunner:  logic.NewProcessRunner(),
		LLMProvider:    provider,
		LLMModel:       model,
		LLMKey:         key,
        AIResponseHistory:    make([]AIHistoryEntry, 0),
		AIHistoryContextSize: 3,
		AIUseOpenTabsAsContext: false,
	}
	ew.TabManager = NewTabManager(ew)
	return ew
}

func (e *EditorWindow) SetupUI() {
	e.Window.SetWindowTitle("Go Lite IDE")
	e.Window.Resize2(1024, 768)

	e.Window.ConnectCloseEvent(func(event *gui.QCloseEvent) {
		// Проверяем наличие несохранённых изменений
		if e.TabManager.HasUnsavedChanges() {
			// Показываем единый диалог для всех несохранённых файлов
			if !e.TabManager.PromptSaveAll() {
				event.Ignore()
				return
			}
		}
		
		// Останавливаем запущенные процессы
		if e.ProcessRunner != nil {
			e.ProcessRunner.StopAll()
		}
		
		event.Accept()
	})

	// 1. Central Widget is the Tab Manager
	e.Window.SetCentralWidget(e.TabManager.Tabs)

	// 2. Setup Docks
	e.setupProjectDock()
	e.setupOutputDock()
	e.setupAIDock()

	// 3. Menus
	e.createMenus()

	// 4. Status Bar
	e.Window.StatusBar().ShowMessage("Ready", 0)

	// 5. Global Keyboard Shortcuts (Escape to close search)
	e.setupGlobalShortcuts()

}

func (e *EditorWindow) setupProjectDock() {
	e.ProjectTree = NewProjectTreeWidget(e)
	e.Window.AddDockWidget(core.Qt__LeftDockWidgetArea, e.ProjectTree.DockWidget)
	e.ProjectTree.DockWidget.SetVisible(false) 
}

func (e *EditorWindow) setupOutputDock() {
	e.OutputDock = widgets.NewQDockWidget("Terminal / Run Output", e.Window, 0)
	e.OutputDock.SetObjectName("OutputDock")
	
	wrapper := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.SetContentsMargins(0,0,0,0)

	// Toolbar
	toolbar := widgets.NewQHBoxLayout()
	btnClear := widgets.NewQPushButton2("Clear", nil)
	btnClear.ConnectClicked(func(bool) { e.OutputText.Clear() })
	
	// Stop Button
	e.BtnStop = widgets.NewQPushButton2("Stop Process", nil)
	e.BtnStop.SetStyleSheet("color: red; font-weight: bold;")
	e.BtnStop.SetEnabled(false)

	toolbar.AddWidget(btnClear, 0, 0)
	toolbar.AddWidget(e.BtnStop, 0, 0)
	toolbar.AddStretch(1)

	layout.AddLayout(toolbar, 0)

	e.OutputText = widgets.NewQPlainTextEdit(nil)
	e.OutputText.SetReadOnly(true)
	// NEW: Используем цвета текущей схемы
	scheme := e.TabManager.CurrentScheme
	if scheme != nil {
		e.OutputText.SetStyleSheet(fmt.Sprintf(
			"background-color: %s; color: %s; font-family: Monospace;",
			scheme.Background, scheme.Foreground))
	} else {
		e.OutputText.SetStyleSheet("background-color: #1e1e1e; color: #d4d4d4; font-family: Monospace;")
	}

	layout.AddWidget(e.OutputText, 0, 0)

	wrapper.SetLayout(layout)
	e.OutputDock.SetWidget(wrapper)

	e.Window.AddDockWidget(core.Qt__BottomDockWidgetArea, e.OutputDock)
	e.OutputDock.Hide()
}

func (e *EditorWindow) setupAIDock() {
	e.AIDock = widgets.NewQDockWidget("AI Assistant", e.Window, 0)
	
	wrapper := widgets.NewQWidget(nil, 0)
	layout := widgets.NewQVBoxLayout()
	layout.SetContentsMargins(5, 5, 5, 5)
	layout.SetSpacing(5)

	// Секция отображения контекста ===
	contextGroup := widgets.NewQGroupBox2("Context Files", nil)
	contextLayout := widgets.NewQVBoxLayout()
	contextLayout.SetContentsMargins(5, 5, 5, 5)
	
	e.AIContextLabel = widgets.NewQLabel2("No context files selected", nil, 0)
	e.AIContextLabel.SetWordWrap(true)
	e.AIContextLabel.SetStyleSheet("color: #888; font-size: 11px;")
	e.AIContextLabel.SetMaximumHeight(80)
	contextLayout.AddWidget(e.AIContextLabel, 0, 0)
	
	contextGroup.SetLayout(contextLayout)
	contextGroup.SetMaximumHeight(120)
	layout.AddWidget(contextGroup, 0, 0)

    // Chat History
	e.AIChat = widgets.NewQTextBrowser(nil)
	e.AIChat.SetOpenExternalLinks(false)
	e.AIChat.SetReadOnly(true)
	e.AIChat.SetTextInteractionFlags(
		core.Qt__TextBrowserInteraction | core.Qt__LinksAccessibleByMouse,
	) // разрешаем только клик по ссылкам
	
    // Важно: обрабатываем клик строго в UI потоке и запрещаем "навигацию" QTextBrowser.
    e.AIChat.ConnectAnchorClicked(func(link *core.QUrl) {
    	if link == nil {
    		return
    	}
    	urlStr := link.ToString(core.QUrl__None)
    	e.handleCodeBlockClick(urlStr)
    	e.AIChat.SetSource(core.NewQUrl())
    })
    
	layout.AddWidget(e.AIChat, 1, 0)
	optionsLayout := widgets.NewQHBoxLayout()
	optionsLayout.SetContentsMargins(0, 0, 0, 0)
	
	e.AIClipboardCheckbox = widgets.NewQCheckBox2("Include clipboard", nil)
	e.AIClipboardCheckbox.SetToolTip("Add clipboard content as context for AI")
	e.AIClipboardCheckbox.SetChecked(false)

	optionsLayout.AddWidget(e.AIClipboardCheckbox, 0, 0)
	optionsLayout.AddStretch(1) // Добавляем разделитель перед кнопками

	// Кнопка очистки контекста
	btnClearContext := widgets.NewQPushButton2("Clear Context", nil)
	// Обновляем подсказку, чтобы она включала историю чата
	btnClearContext.SetToolTip("Clear all optional context:\n- Project files\n- Other open tabs\n- Clipboard\n- Chat history")
	btnClearContext.ConnectClicked(func(bool) {
		// 1. Очищаем файлы проекта из контекста
		e.ProjectManager.ClearContextFiles()

		// 2. Отключаем контекст из других вкладок
		e.AIUseOpenTabsAsContext = false
		if e.actUseTabsContext != nil {
			e.actUseTabsContext.SetChecked(false)
		}

		// 3. Снимаем галочку с буфера обмена
		if e.AIClipboardCheckbox != nil {
			e.AIClipboardCheckbox.SetChecked(false)
		}
		
		// 4. (ИСПРАВЛЕНИЕ) Очищаем историю диалога
		e.ClearAIHistory()

		// 5. Обновляем UI и показываем сообщение
		e.UpdateAIContextDisplay()
		e.Window.StatusBar().ShowMessage("All AI context has been cleared", 2000)
	})
	optionsLayout.AddWidget(btnClearContext, 0, 0)

	// Кнопка обновления
	btnRefreshContext := widgets.NewQPushButton2("↻", nil)
	btnRefreshContext.SetToolTip("Refresh context display")
	btnRefreshContext.SetMaximumWidth(30)
	btnRefreshContext.ConnectClicked(func(bool) {
		e.UpdateAIContextDisplay()
	})
	optionsLayout.AddWidget(btnRefreshContext, 0, 0)

	optionsLayout.AddStretch(1)
	layout.AddLayout(optionsLayout, 0)

	// Input Area
	e.AIInput = widgets.NewQPlainTextEdit(nil)
	e.AIInput.SetPlaceholderText("Ask AI about your code ...")
	e.AIInput.SetMaximumHeight(100)
	layout.AddWidget(e.AIInput, 0, 0)

	// Send Button
	btnSend := widgets.NewQPushButton2("Send", nil)
	layout.AddWidget(btnSend, 0, 0)

	wrapper.SetLayout(layout)
	e.AIDock.SetWidget(wrapper)

	e.Window.AddDockWidget(core.Qt__RightDockWidgetArea, e.AIDock)
	e.AIDock.Hide()

	// Connect Send
	sendFunc := func() {
		text := e.AIInput.ToPlainText()
		if text == "" { return }
		e.AIInput.Clear()
		e.HandleAskLLM(text)
	}
	btnSend.ConnectClicked(func(bool) { sendFunc() })
	
	// Обновляем контекст при открытии панели ===
	e.AIDock.ConnectVisibilityChanged(func(visible bool) {
		if visible {
			e.UpdateAIContextDisplay()
		}
	})
}

func (e *EditorWindow) OpenPath(path string) {
	stat, err := os.Stat(path)
	if err != nil { 
		e.Window.StatusBar().ShowMessage(fmt.Sprintf("Error: %v", err), 3000)
		return 
	}

	if stat.IsDir() {
		// Логика открытия проекта
		e.ProjectManager.SetRootPath(path)
		e.ProjectTree.Refresh()
		e.ProjectTree.DockWidget.Show()
		
		// Обновляем заголовок окна
		e.Window.SetWindowTitle(fmt.Sprintf("%s - Go Lite IDE", filepath.Base(path)))
		e.Window.StatusBar().ShowMessage(fmt.Sprintf("Project opened: %s", path), 3000)
	} else {
		// Логика открытия одиночного файла
		e.TabManager.OpenFile(path)
		// Если проект не активен, можно обновить заголовок по файлу
		if !e.ProjectManager.IsActive {
			e.Window.SetWindowTitle(fmt.Sprintf("%s - Go Lite IDE", filepath.Base(path)))
		}
	}
}

func (e *EditorWindow) RunOnUIThread(f func()) {
	timer := core.NewQTimer(e.Window)
	timer.SetSingleShot(true)
	timer.ConnectTimeout(f)
	timer.Start(0)
}

// setupGlobalShortcuts настраивает глобальные горячие клавиши
func (e *EditorWindow) setupGlobalShortcuts() {

	// Escape — закрывает панель поиска, отклоняет предложение или очищает подсветку скобок
	escShortcut := widgets.NewQShortcut(e.Window)
	escShortcut.SetKey(gui.NewQKeySequence2("Escape", gui.QKeySequence__NativeText))
	escShortcut.SetContext(core.Qt__WidgetWithChildrenShortcut)
	escShortcut.ConnectActivated(func() {
		if ed := e.TabManager.CurrentEditor(); ed != nil {
			// Приоритет 1: Очищаем подсветку скобок
			if ed.BracketHighlightActive {
				e.TabManager.ClearBracketHighlight(ed)
				return
			}
			// Приоритет 2: Отклоняем предложение AI
			if ed.HasSuggestion {
				e.TabManager.RejectSuggestion(ed)
				return
			}
			// Приоритет 3: Закрываем панель поиска
			if ed.SearchWidget != nil && ed.SearchWidget.IsVisible() {
				ed.SearchWidget.Hide()
			}
		}
	})

    // Ctrl+Space — принудительный вызов однострочного автодополнения (альтернатива Tab)
    completeShortcut := widgets.NewQShortcut(e.Window)
    completeShortcut.SetKey(gui.NewQKeySequence2("Ctrl+Space", gui.QKeySequence__NativeText))
    completeShortcut.SetContext(core.Qt__WidgetWithChildrenShortcut)
    completeShortcut.ConnectActivated(func() {
        if e.TabManager.IsLineCompleteEnabled() {
            e.TabManager.TriggerLineComplete()
        } else {
            e.Window.StatusBar().ShowMessage("AI Line Completion is disabled. Enable it in Edit menu.", 3000)
        }
    })
    
    // Ctrl+L — многострочное автодополнение (полный код)
    multiLineShortcut := widgets.NewQShortcut(e.Window)
    multiLineShortcut.SetKey(gui.NewQKeySequence2("Ctrl+L", gui.QKeySequence__NativeText))
    multiLineShortcut.SetContext(core.Qt__WidgetWithChildrenShortcut)
    multiLineShortcut.ConnectActivated(func() {
        if e.TabManager.IsAutoCompleteEnabled() {
            e.TabManager.TriggerAutoComplete()
        } else {
            e.Window.StatusBar().ShowMessage("AI Code Completion is disabled. Enable it in Edit menu.", 3000)
        }
    })
    // Ctrl+Shift+L — открыть AI Assistant (канвас диалога с LLM), без изменений меню
    openAICanvasShortcut := widgets.NewQShortcut(e.Window)
    openAICanvasShortcut.SetKey(gui.NewQKeySequence2("Ctrl+Shift+L", gui.QKeySequence__NativeText))
    openAICanvasShortcut.SetContext(core.Qt__WidgetWithChildrenShortcut)
    openAICanvasShortcut.ConnectActivated(func() {
        // Показать док и обновить отображение контекста
        e.AIDock.Show()
        e.UpdateAIContextDisplay()

        // Фокус в поле ввода, чтобы можно было сразу печатать
        if e.AIInput != nil {
            e.AIInput.SetFocus2()
        }
        e.Window.StatusBar().ShowMessage("AI Assistant opened", 1500)
    })
   
}

// Show отображает главное окно
func (e *EditorWindow) Show() {
	e.Window.Show()
}


// UpdateAIContextDisplay обновляет отображение списка файлов контекста на AI панели
func (e *EditorWindow) UpdateAIContextDisplay() {
	if e.AIContextLabel == nil {
		return
	}

	var contextParts []string

	// 1. Текущий открытый файл
	if ed := e.TabManager.CurrentEditor(); ed != nil && ed.FilePath != "" {
		contextParts = append(contextParts, fmt.Sprintf("📄 <b>Current:</b> %s", filepath.Base(ed.FilePath)))
	} else if ed != nil {
		contextParts = append(contextParts, "📄 <b>Current:</b> Untitled (unsaved)")
	}

	// 2. Файлы из контекста проекта
	projectFiles := e.ProjectManager.GetContextFiles()
	if len(projectFiles) > 0 {
		contextParts = append(contextParts, fmt.Sprintf("📁 <b>Project context:</b> %d file(s)", len(projectFiles)))
		// Показываем первые 3 файла
		for i, f := range projectFiles {
			if i >= 3 {
				contextParts = append(contextParts, fmt.Sprintf("   ... and %d more", len(projectFiles)-3))
				break
			}
			contextParts = append(contextParts, fmt.Sprintf("   • %s", filepath.Base(f)))
		}
	}

	// 2.5 Контекст из других открытых вкладок
	if e.AIUseOpenTabsAsContext {
		ed := e.TabManager.CurrentEditor()
		_, tabNames := e.TabManager.GetAllOpenTabsContext(ed)
		if len(tabNames) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("🧩 <b>Open tabs context:</b> %d tab(s)", len(tabNames)))
			// Показать первые 3 вкладки
			for i, name := range tabNames {
				if i >= 3 {
					contextParts = append(contextParts, fmt.Sprintf("   ... and %d more", len(tabNames)-3))
					break
				}
				contextParts = append(contextParts, fmt.Sprintf("   • %s", name))
			}
		} else {
			contextParts = append(contextParts, "🧩 <b>Open tabs context:</b> enabled (no other tabs with content)")
		}
	} else {
		contextParts = append(contextParts, "🧩 <b>Open tabs context:</b> disabled")
	}


	// 3. Статус буфера обмена
	if e.AIClipboardCheckbox != nil && e.AIClipboardCheckbox.IsChecked() {
		clipboard := gui.QGuiApplication_Clipboard()
		clipText := clipboard.Text(gui.QClipboard__Clipboard)

		if clipText != "" {
			// Показываем превью (первые 50 символов)
			preview := clipText
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			// Экранируем HTML
			preview = strings.ReplaceAll(preview, "<", "&lt;")
			preview = strings.ReplaceAll(preview, ">", "&gt;")
			preview = strings.ReplaceAll(preview, "\n", " ")
			contextParts = append(contextParts, fmt.Sprintf("📋 <b>Clipboard:</b> \"%s\"", preview))
		} else {
			contextParts = append(contextParts, "📋 <b>Clipboard:</b> (empty)")
		}
	}

	// 4. История диалога с AI
	if e.AIHistoryContextSize > 0 {
		historyCount := len(e.AIResponseHistory)
		usedCount := e.AIHistoryContextSize
		if usedCount > historyCount {
			usedCount = historyCount
		}
		contextParts = append(contextParts, 
			fmt.Sprintf("💬 <b>Chat history:</b> %d of %d (max: %d)", 
				usedCount, historyCount, e.AIHistoryContextSize))
	} else {
		contextParts = append(contextParts, "💬 <b>Chat history:</b> disabled")
	}

	// Формируем итоговый текст
	if len(contextParts) == 0 {
		e.AIContextLabel.SetText("No context files selected")
		e.AIContextLabel.SetStyleSheet("color: #888; font-size: 11px;")
	} else {
		e.AIContextLabel.SetText(strings.Join(contextParts, "<br>"))
		e.AIContextLabel.SetStyleSheet("color: #aaa; font-size: 11px;")
	}
}

// AddToAIHistory добавляет запись в историю диалога с AI
func (e *EditorWindow) AddToAIHistory(userPrompt, aiResponse string) {
	entry := AIHistoryEntry{
		UserPrompt: userPrompt,
		AIResponse: aiResponse,
		Timestamp:  time.Now().Format("15:04:05"),
	}
	
	e.AIResponseHistory = append(e.AIResponseHistory, entry)
	
	// Ограничиваем размер истории (храним максимум 50 записей)
	const maxHistorySize = 50
	if len(e.AIResponseHistory) > maxHistorySize {
		e.AIResponseHistory = e.AIResponseHistory[len(e.AIResponseHistory)-maxHistorySize:]
	}
}

// GetAIHistoryContext возвращает последние N ответов для использования как контекст
func (e *EditorWindow) GetAIHistoryContext() string {
	if e.AIHistoryContextSize <= 0 || len(e.AIResponseHistory) == 0 {
		return ""
	}
	
	// Определяем, сколько записей взять
	count := e.AIHistoryContextSize
	if count > len(e.AIResponseHistory) {
		count = len(e.AIResponseHistory)
	}
	
	// Берём последние N записей
	startIdx := len(e.AIResponseHistory) - count
	relevantHistory := e.AIResponseHistory[startIdx:]
	
	var sb strings.Builder
	sb.WriteString("\n--- Previous conversation context ---\n")
	
	for i, entry := range relevantHistory {
		sb.WriteString(fmt.Sprintf("\n[%d] User asked:\n%s\n", i+1, truncateForContext(entry.UserPrompt, 500)))
		sb.WriteString(fmt.Sprintf("\n[%d] AI responded:\n%s\n", i+1, truncateForContext(entry.AIResponse, 1500)))
	}
	
	sb.WriteString("--- End of previous conversation ---\n")
	
	return sb.String()
}

// ClearAIHistory очищает историю диалога
func (e *EditorWindow) ClearAIHistory() {
	e.AIResponseHistory = make([]AIHistoryEntry, 0)
	e.Window.StatusBar().ShowMessage("AI conversation history cleared", 2000)
}

// truncateForContext обрезает текст до указанной длины для использования в контексте
func truncateForContext(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "\n... [truncated]"
}
