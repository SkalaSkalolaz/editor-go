package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/ui"
)

func main() {
	const appVersion = "0.9.1"
	// Объявляем переменные для флагов
	var (
		provider    string
		model       string
		apiKey      string
		showHelp    bool
		showVersion bool
	)

	// Привязываем длинные и короткие флаги к одним и тем же переменным
	flag.StringVar(&provider, "provider", "ollama", "LLM Provider (ollama, openrouter, pollinations, URL provider)")
	flag.StringVar(&model, "model", "gemma:2b", "LLM Model name")
	flag.StringVar(&apiKey, "key", "", "API Key (if required)")

	// Help: поддерживаем и -h, и --help
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")

	// Version: поддерживаем и -v, и --version
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Go Lite IDE v%s\n", appVersion)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [file_or_dir]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Обработка флагов немедленного действия
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("Go Lite IDE version %s\n", appVersion)
		os.Exit(0)
	}

	// Оставшиеся аргументы считаем путем к файлу или проекту
	initialPath := ""
	if flag.NArg() > 0 {
		initialPath = flag.Arg(0)
	}

	// 2. Инициализация Qt
	app := widgets.NewQApplication(len(os.Args), os.Args)
	app.SetApplicationName("Go Lite IDE")
	app.SetApplicationVersion(appVersion) // Используем константу

	// 3. Создание главного окна
	mainWindow := ui.NewEditorWindow(provider, model, apiKey)
	mainWindow.SetupUI()

	// 4. Открытие начального пути (если есть)
	if initialPath != "" {
		mainWindow.OpenPath(initialPath)
	} else {
		// Если ничего не открыто, создаем пустую вкладку
		mainWindow.TabManager.NewTab()
	}

	mainWindow.Show()
	app.Exec()
}
