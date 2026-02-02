package ui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
)

// hexToQColor конвертирует HEX строку в QColor
func hexToQColor(hex string) *gui.QColor {
	// Убираем # если есть
	hex = strings.TrimPrefix(hex, "#")
	
	// Парсим RGB компоненты
	r, _ := strconv.ParseInt(hex[0:2], 16, 64)
	g, _ := strconv.ParseInt(hex[2:4], 16, 64)
	b, _ := strconv.ParseInt(hex[4:6], 16, 64)
	
	return gui.NewQColor3(int(r), int(g), int(b), 255)
}

// ColorScheme определяет цветовую схему подсветки
type ColorScheme struct {
	Name            string
	Background      string
	Foreground      string
	Keyword         string
	Type            string
	String          string
	Comment         string
	Number          string
	Function        string
	Operator        string
	CurrentLine     string
}

// Предопределённые схемы
var ColorSchemes = map[string]*ColorScheme{
	"Monokai": {
		Name:        "Monokai",
		Background:  "#272822",
		Foreground:  "#F8F8F2",
		Keyword:     "#F92672",
		Type:        "#66D9EF",
		String:      "#E6DB74",
		Comment:     "#75715E",
		Number:      "#AE81FF",
		Function:    "#A6E22E",
		Operator:    "#F92672",
		CurrentLine: "#3E3D32",
	},
	"Dracula": {
		Name:        "Dracula",
		Background:  "#282A36",
		Foreground:  "#F8F8F2",
		Keyword:     "#FF79C6",
		Type:        "#8BE9FD",
		String:      "#F1FA8C",
		Comment:     "#6272A4",
		Number:      "#BD93F9",
		Function:    "#50FA7B",
		Operator:    "#FF79C6",
		CurrentLine: "#44475A",
	},
	"One Dark": {
		Name:        "One Dark",
		Background:  "#282C34",
		Foreground:  "#ABB2BF",
		Keyword:     "#C678DD",
		Type:        "#E5C07B",
		String:      "#98C379",
		Comment:     "#5C6370",
		Number:      "#D19A66",
		Function:    "#61AFEF",
		Operator:    "#56B6C2",
		CurrentLine: "#2C323C",
	},
	"Solarized Dark": {
		Name:        "Solarized Dark",
		Background:  "#002B36",
		Foreground:  "#839496",
		Keyword:     "#859900",
		Type:        "#B58900",
		String:      "#2AA198",
		Comment:     "#586E75",
		Number:      "#D33682",
		Function:    "#268BD2",
		Operator:    "#859900",
		CurrentLine: "#073642",
	},
	"GitHub Dark": {
		Name:        "GitHub Dark",
		Background:  "#0D1117",
		Foreground:  "#C9D1D9",
		Keyword:     "#FF7B72",
		Type:        "#FFA657",
		String:      "#A5D6FF",
		Comment:     "#8B949E",
		Number:      "#79C0FF",
		Function:    "#D2A8FF",
		Operator:    "#FF7B72",
		CurrentLine: "#161B22",
	},
}

// HighlightRule определяет правило подсветки
type HighlightRule struct {
	Pattern *regexp.Regexp
	Format  *gui.QTextCharFormat
}

// MultiLineRule для многострочных комментариев и строк
type MultiLineRule struct {
	StartPattern *regexp.Regexp
	EndPattern   *regexp.Regexp
	Format       *gui.QTextCharFormat
	StateID      int
}

type LanguageDefinition struct {
	Keywords     []string
	Types        []string
	Operators    string
	SingleLine   string 
	MultiLineIn  string 
	MultiLineOut string 
}

var Languages = map[string]LanguageDefinition{
	"go": {
		Keywords:     []string{"break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var"},
		Types:        []string{"bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "true", "false", "nil", "iota"},
		Operators:    `(:=|&&|\|\||<-|->|<<|>>|<=|>=|==|!=|\+\+|--|[+\-*/%&|^!<>=])`,
		SingleLine:   `//`,
		MultiLineIn:  `/\*`,
		MultiLineOut: `\*/`,
	},
	"javascript": {
		Keywords:     []string{"break", "case", "catch", "class", "const", "continue", "debugger", "default", "delete", "do", "else", "export", "extends", "finally", "for", "function", "if", "import", "in", "instanceof", "new", "return", "super", "switch", "this", "throw", "try", "typeof", "var", "void", "while", "with", "yield", "let", "static", "enum", "await", "async"},
		Types:        []string{"null", "undefined", "true", "false", "NaN", "Infinity", "window", "document", "console", "Object", "Array", "String", "Number", "Boolean", "Promise"},
		Operators:    `(=>|\+\+|--|===|!==|&&|\|\||[+\-*/%&|^!<>=]=?)`,
		SingleLine:   `//`,
		MultiLineIn:  `/\*`,
		MultiLineOut: `\*/`,
	},
	"html": {
		Keywords:     []string{"!DOCTYPE", "html", "head", "title", "body", "header", "footer", "nav", "section", "article", "aside", "h1", "h2", "h3", "h4", "h5", "h6", "div", "span", "p", "br", "hr", "a", "img", "ul", "ol", "li", "table", "tr", "td", "th", "form", "input", "button", "select", "option", "textarea", "label", "script", "style", "meta", "link"},
		Types:        []string{"id", "class", "src", "href", "type", "value", "name", "placeholder", "style", "rel", "alt", "width", "height", "onclick", "onload"},
		Operators:    `([<>!/=-])`,
		SingleLine:   "", 
		MultiLineIn:  `<!--`,
		MultiLineOut: `-->`,
	},
}

type UniversalSyntaxHighlighter struct {
	*gui.QSyntaxHighlighter
	rules          []*HighlightRule
	multiLineRules []*MultiLineRule
	scheme         *ColorScheme
	lang           LanguageDefinition

	keywordFormat  *gui.QTextCharFormat
	typeFormat     *gui.QTextCharFormat
	stringFormat   *gui.QTextCharFormat
	commentFormat  *gui.QTextCharFormat
	numberFormat   *gui.QTextCharFormat
	functionFormat *gui.QTextCharFormat
	operatorFormat *gui.QTextCharFormat
}

func NewUniversalHighlighter(parent *gui.QTextDocument, langName string, scheme *ColorScheme) *UniversalSyntaxHighlighter {
	if scheme == nil { scheme = ColorSchemes["Monokai"] }
	lang, ok := Languages[langName]
	if !ok { lang = Languages["go"] }

	h := &UniversalSyntaxHighlighter{
		QSyntaxHighlighter: gui.NewQSyntaxHighlighter2(parent),
		scheme:             scheme,
		lang:               lang,
	}
	h.setupFormats()
	h.setupRules()
	h.ConnectHighlightBlock(h.highlightBlock)
    h.Rehighlight() 
    
	return h
}

func (h *UniversalSyntaxHighlighter) setupFormats() {
	h.keywordFormat = gui.NewQTextCharFormat()
	h.keywordFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Keyword), core.Qt__SolidPattern))
	h.keywordFormat.SetFontWeight(int(gui.QFont__Bold))

	h.typeFormat = gui.NewQTextCharFormat()
	h.typeFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Type), core.Qt__SolidPattern))

	h.stringFormat = gui.NewQTextCharFormat()
	h.stringFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.String), core.Qt__SolidPattern))

	h.commentFormat = gui.NewQTextCharFormat()
	h.commentFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Comment), core.Qt__SolidPattern))
	h.commentFormat.SetFontItalic(true)

	h.numberFormat = gui.NewQTextCharFormat()
	h.numberFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Number), core.Qt__SolidPattern))

	h.functionFormat = gui.NewQTextCharFormat()
	h.functionFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Function), core.Qt__SolidPattern))

	h.operatorFormat = gui.NewQTextCharFormat()
	h.operatorFormat.SetForeground(gui.NewQBrush3(hexToQColor(h.scheme.Operator), core.Qt__SolidPattern))
}

func (h *UniversalSyntaxHighlighter) setupRules() {
	h.rules = make([]*HighlightRule, 0)

	// Keywords
	keywordPattern := `\b(` + strings.Join(h.lang.Keywords, "|") + `)\b`
	if h.lang.MultiLineIn == "<!--" { // HTML
		keywordPattern = `(<[ /]*|/?>|` + strings.Join(h.lang.Keywords, "|") + `)`
	}
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(keywordPattern),
		Format:  h.keywordFormat,
	})

	// Types/Attributes
	typePattern := `\b(` + strings.Join(h.lang.Types, "|") + `)\b`
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(typePattern),
		Format:  h.typeFormat,
	})

	// Numbers
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(`\b\d+\b`),
		Format:  h.numberFormat,
	})

	// Functions
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(`\b[A-Za-z0-9_]+\(`),
		Format:  h.functionFormat,
	})

	// Operators
	if h.lang.Operators != "" {
		h.rules = append(h.rules, &HighlightRule{
			Pattern: regexp.MustCompile(h.lang.Operators),
			Format:  h.operatorFormat,
		})
	}

	// Single-line comments
	if h.lang.SingleLine != "" {
		h.rules = append(h.rules, &HighlightRule{
			Pattern: regexp.MustCompile(regexp.QuoteMeta(h.lang.SingleLine) + `[^
]*`),
			Format:  h.commentFormat,
		})
	}

	// Strings
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(`"(?:[^"\\]|\\.)*"`),
		Format:  h.stringFormat,
	})
	h.rules = append(h.rules, &HighlightRule{
		Pattern: regexp.MustCompile(`'(?:[^'\\]|\\.)*'`),
		Format:  h.stringFormat,
	})

	// Multiline
	if h.lang.MultiLineIn != "" {
		h.multiLineRules = []*MultiLineRule{
			{
				StartPattern: regexp.MustCompile(regexp.QuoteMeta(h.lang.MultiLineIn)),
				EndPattern:   regexp.MustCompile(regexp.QuoteMeta(h.lang.MultiLineOut)),
				Format:       h.commentFormat,
				StateID:      1,
			},
		}
	}
}

// highlightBlock вызывается Qt для каждого блока текста
func (h *UniversalSyntaxHighlighter) highlightBlock(text string) {
	// Обрабатываем многострочные конструкции
	h.handleMultiLineHighlight(text)

	// Применяем однострочные правила
	for _, rule := range h.rules {
		matches := rule.Pattern.FindAllStringIndex(text, -1)
		for _, match := range matches {
			start := match[0]
			length := match[1] - match[0]

			// Для функций нам нужно исключить скобку из подсветки
			if rule.Format == h.functionFormat {
				// Находим только имя функции (без скобки)
				funcText := text[start:match[1]]
				parenIdx := strings.Index(funcText, "(")
				if parenIdx > 0 {
					length = parenIdx
				}
			}

			h.SetFormat(start, length, rule.Format)
		}
	}
}

// handleMultiLineHighlight обрабатывает многострочные комментарии и строки
func (h *UniversalSyntaxHighlighter) handleMultiLineHighlight(text string) {
	previousState := h.PreviousBlockState()

	for _, rule := range h.multiLineRules {
		startIndex := 0

		if previousState != rule.StateID {
			// Ищем начало многострочной конструкции
			loc := rule.StartPattern.FindStringIndex(text)
			if loc != nil {
				startIndex = loc[0]
			} else {
				continue
			}
		}

		for startIndex >= 0 {
			var endIndex int
			var matchLength int

			// Ищем конец конструкции
			endLoc := rule.EndPattern.FindStringIndex(text[startIndex:])
			if endLoc != nil && (previousState != rule.StateID || endLoc[0] > 0) {
				actualEnd := startIndex + endLoc[1]
				endIndex = actualEnd
				matchLength = endIndex - startIndex
				h.SetCurrentBlockState(-1)
			} else {
				// Конец не найден — подсвечиваем до конца строки
				h.SetCurrentBlockState(rule.StateID)
				matchLength = len(text) - startIndex
			}

			h.SetFormat(startIndex, matchLength, rule.Format)

			// Ищем следующее вхождение
			nextLoc := rule.StartPattern.FindStringIndex(text[startIndex+matchLength:])
			if nextLoc != nil {
				startIndex = startIndex + matchLength + nextLoc[0]
			} else {
				startIndex = -1
			}
		}
	}
}

// SetScheme меняет цветовую схему
func (h *UniversalSyntaxHighlighter) SetScheme(scheme *ColorScheme) {
	h.scheme = scheme
	h.setupFormats()
	h.setupRules()
	h.Rehighlight()
}

// GetScheme возвращает текущую схему
func (h *UniversalSyntaxHighlighter) GetScheme() *ColorScheme {
	return h.scheme
}

// GetBracketHighlightColor возвращает контрастный цвет для подсветки скобок
// в зависимости от текущей цветовой схемы
func GetBracketHighlightColor(scheme *ColorScheme) *gui.QColor {
	if scheme == nil {
		return gui.NewQColor3(255, 215, 0, 100) // Золотой по умолчанию
	}

	// Подбираем контрастные цвета для каждой схемы
	bracketColors := map[string]*gui.QColor{
		"Monokai":        gui.NewQColor3(255, 215, 0, 80),   // Золотой
		"Dracula":        gui.NewQColor3(139, 233, 253, 80), // Голубой
		"One Dark":       gui.NewQColor3(224, 108, 117, 80), // Красноватый
		"Solarized Dark": gui.NewQColor3(181, 137, 0, 80),   // Жёлтый
		"GitHub Dark":    gui.NewQColor3(255, 123, 114, 80), // Коралловый
	}

	if color, ok := bracketColors[scheme.Name]; ok {
		return color
	}

	return gui.NewQColor3(255, 215, 0, 80) // Fallback
}
