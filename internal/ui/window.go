package ui

import (
	"os"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
    "github.com/therecipe/qt/gui"

	"go-gnome-editor/internal/logic"
)

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

    // Runtime Configuration
	RunArgs string

	// Logic State
	LLMProvider string
	LLMModel    string
	LLMKey      string
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
	}
	ew.TabManager = NewTabManager(ew)
	return ew
}

func (e *EditorWindow) SetupUI() {
	e.Window.SetWindowTitle("Go Lite IDE")
	e.Window.Resize2(1024, 768)

	e.Window.ConnectCloseEvent(func(event *gui.QCloseEvent) {
		// Пробуем закрыть все вкладки с конца (чтобы индексы не сбивались слишком сильно, хотя CloseTab ищет по виджету)
		// Если пользователь нажмет Cancel в любой вкладке, отменяем закрытие программы.
		count := e.TabManager.Tabs.Count()
		for i := count - 1; i >= 0; i-- {
			// Вызываем CloseTab для каждой вкладки. Если вернул false (Cancel), прерываем выход.
			if !e.TabManager.CloseTab(i) {
				event.Ignore()
				return
			}
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
	e.OutputText.SetStyleSheet("background-color: #1e1e1e; color: #d4d4d4; font-family: Monospace;")
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

	e.AIChat = widgets.NewQTextBrowser(nil)
	e.AIChat.SetOpenExternalLinks(true)
	layout.AddWidget(e.AIChat, 1, 0)

	e.AIInput = widgets.NewQPlainTextEdit(nil)
	e.AIInput.SetPlaceholderText("Ask AI about your code (Ctrl+Enter to send)...")
	e.AIInput.SetMaximumHeight(100)
	layout.AddWidget(e.AIInput, 0, 0)

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
}

func (e *EditorWindow) OpenPath(path string) {
	stat, err := os.Stat(path)
	if err != nil { return }

	if stat.IsDir() {
		e.ProjectManager.SetRootPath(path)
		e.ProjectTree.Refresh()
		e.ProjectTree.DockWidget.Show()
		e.Window.SetWindowTitle(path + " - Go Lite IDE")
	} else {
		e.TabManager.OpenFile(path)
	}
}

func (e *EditorWindow) RunOnUIThread(f func()) {
	timer := core.NewQTimer(e.Window)
	timer.SetSingleShot(true)
	timer.ConnectTimeout(f)
	timer.Start(0)
}

func (e *EditorWindow) Show() {
	e.Window.Show()
}

// setupGlobalShortcuts настраивает глобальные горячие клавиши
func (e *EditorWindow) setupGlobalShortcuts() {
	// Escape — закрывает панель поиска
	// Используем QShortcut с правильной сигнатурой для therecipe/qt
	escShortcut := widgets.NewQShortcut(e.Window)
	escShortcut.SetKey(gui.NewQKeySequence2("Escape", gui.QKeySequence__NativeText))
	escShortcut.SetContext(core.Qt__WidgetWithChildrenShortcut)
	escShortcut.ConnectActivated(func() {
		if ed := e.TabManager.CurrentEditor(); ed != nil && ed.SearchWidget != nil {
			if ed.SearchWidget.IsVisible() {
				ed.SearchWidget.Hide()
			}
		}
	})
}
