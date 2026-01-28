package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/therecipe/qt/widgets"
	"go-gnome-editor/internal/ui"
)

func main() {
	// 1. Используем flag для чистого парсинга аргументов
	provider := flag.String("provider", "ollama", "LLM Provider (ollama, openrouter, pollinations)")
	model := flag.String("model", "gemma:2b", "LLM Model name")
	apiKey := flag.String("key", "", "API Key (if required)")
	help := flag.Bool("help", false, "Show help")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [file_or_dir]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *help {
		flag.Usage()
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
	app.SetApplicationVersion("1.0.0")

	// 3. Создание главного окна
	mainWindow := ui.NewEditorWindow(*provider, *model, *apiKey)
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
