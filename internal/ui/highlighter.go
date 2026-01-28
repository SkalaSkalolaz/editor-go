package ui

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/therecipe/qt/gui"
)

// SyntaxHighlighter обертка над Qt хайлайтером с использованием Chroma
type SyntaxHighlighter struct {
	*gui.QSyntaxHighlighter

	lexer   chroma.Lexer
	formats map[chroma.TokenType]*gui.QTextCharFormat
}

// NewSyntaxHighlighter создает и привязывает хайлайтер к документу
func NewSyntaxHighlighter(parent *gui.QTextDocument) *SyntaxHighlighter {
	h := &SyntaxHighlighter{
		QSyntaxHighlighter: gui.NewQSyntaxHighlighter(parent),
		formats:            make(map[chroma.TokenType]*gui.QTextCharFormat),
	}

	// Инициализируем цветовую схему (стиль "Monokai-подобный")
	h.initFormats()

	// Устанавливаем fallback лексер (plain text)
	h.lexer = lexers.Fallback

	// Переопределяем метод HighlightBlock
	h.ConnectHighlightBlock(func(text string) {
		h.highlightLine(text)
	})

	return h
}

// initFormats создает форматы для разных типов токенов
func (h *SyntaxHighlighter) initFormats() {
	// Ключевые слова
	fmtKeyword := gui.NewQTextCharFormat()
	fmtKeyword.SetForeground(gui.NewQBrush3(gui.NewQColor3(204, 120, 50, 255), 1)) // Orange
	fmtKeyword.SetFontWeight(int(gui.QFont__Bold))

	// Строки
	fmtString := gui.NewQTextCharFormat()
	fmtString.SetForeground(gui.NewQBrush3(gui.NewQColor3(106, 135, 89, 255), 1)) // Green

	// Комментарии
	fmtComment := gui.NewQTextCharFormat()
	fmtComment.SetForeground(gui.NewQBrush3(gui.NewQColor3(128, 128, 128, 255), 1)) // Gray
	fmtComment.SetFontItalic(true)

	// Числа
	fmtNumber := gui.NewQTextCharFormat()
	fmtNumber.SetForeground(gui.NewQBrush3(gui.NewQColor3(104, 151, 187, 255), 1)) // Blue

	// Типы и классы
	fmtType := gui.NewQTextCharFormat()
	fmtType.SetForeground(gui.NewQBrush3(gui.NewQColor3(169, 183, 198, 255), 1)) // Light Blue

	// Функции
	fmtFunction := gui.NewQTextCharFormat()
	fmtFunction.SetForeground(gui.NewQBrush3(gui.NewQColor3(255, 198, 109, 255), 1)) // Yellow

	// Операторы
	fmtOperator := gui.NewQTextCharFormat()
	fmtOperator.SetForeground(gui.NewQBrush3(gui.NewQColor3(249, 38, 114, 255), 1)) // Pink

	// Имена (переменные, идентификаторы)
	fmtName := gui.NewQTextCharFormat()
	fmtName.SetForeground(gui.NewQBrush3(gui.NewQColor3(248, 248, 242, 255), 1)) // White

	// Встроенные функции
	fmtBuiltin := gui.NewQTextCharFormat()
	fmtBuiltin.SetForeground(gui.NewQBrush3(gui.NewQColor3(102, 217, 239, 255), 1)) // Cyan

	// Препроцессор / директивы
	fmtPreproc := gui.NewQTextCharFormat()
	fmtPreproc.SetForeground(gui.NewQBrush3(gui.NewQColor3(174, 129, 255, 255), 1)) // Purple

	// Маппинг токенов Chroma на форматы
	// Keywords
	h.formats[chroma.Keyword] = fmtKeyword
	h.formats[chroma.KeywordConstant] = fmtKeyword
	h.formats[chroma.KeywordDeclaration] = fmtKeyword
	h.formats[chroma.KeywordNamespace] = fmtKeyword
	h.formats[chroma.KeywordPseudo] = fmtKeyword
	h.formats[chroma.KeywordReserved] = fmtKeyword
	h.formats[chroma.KeywordType] = fmtType

	// Strings
	h.formats[chroma.String] = fmtString
	h.formats[chroma.StringAffix] = fmtString
	h.formats[chroma.StringBacktick] = fmtString
	h.formats[chroma.StringChar] = fmtString
	h.formats[chroma.StringDelimiter] = fmtString
	h.formats[chroma.StringDoc] = fmtString
	h.formats[chroma.StringDouble] = fmtString
	h.formats[chroma.StringEscape] = fmtOperator // Escape sequences выделяем иначе
	h.formats[chroma.StringHeredoc] = fmtString
	h.formats[chroma.StringInterpol] = fmtString
	h.formats[chroma.StringOther] = fmtString
	h.formats[chroma.StringRegex] = fmtString
	h.formats[chroma.StringSingle] = fmtString
	h.formats[chroma.StringSymbol] = fmtString

	// Comments
	h.formats[chroma.Comment] = fmtComment
	h.formats[chroma.CommentHashbang] = fmtComment
	h.formats[chroma.CommentMultiline] = fmtComment
	h.formats[chroma.CommentPreproc] = fmtPreproc
	h.formats[chroma.CommentPreprocFile] = fmtPreproc
	h.formats[chroma.CommentSingle] = fmtComment
	h.formats[chroma.CommentSpecial] = fmtComment

	// Numbers
	h.formats[chroma.Number] = fmtNumber
	h.formats[chroma.NumberBin] = fmtNumber
	h.formats[chroma.NumberFloat] = fmtNumber
	h.formats[chroma.NumberHex] = fmtNumber
	h.formats[chroma.NumberInteger] = fmtNumber
	h.formats[chroma.NumberIntegerLong] = fmtNumber
	h.formats[chroma.NumberOct] = fmtNumber

	// Names
	h.formats[chroma.Name] = fmtName
	h.formats[chroma.NameAttribute] = fmtName
	h.formats[chroma.NameBuiltin] = fmtBuiltin
	h.formats[chroma.NameBuiltinPseudo] = fmtBuiltin
	h.formats[chroma.NameClass] = fmtType
	h.formats[chroma.NameConstant] = fmtKeyword
	h.formats[chroma.NameDecorator] = fmtPreproc
	h.formats[chroma.NameEntity] = fmtName
	h.formats[chroma.NameException] = fmtType
	h.formats[chroma.NameFunction] = fmtFunction
	h.formats[chroma.NameFunctionMagic] = fmtFunction
	h.formats[chroma.NameLabel] = fmtName
	h.formats[chroma.NameNamespace] = fmtType
	h.formats[chroma.NameOther] = fmtName
	h.formats[chroma.NameProperty] = fmtName
	h.formats[chroma.NameTag] = fmtKeyword
	h.formats[chroma.NameVariable] = fmtName
	h.formats[chroma.NameVariableClass] = fmtName
	h.formats[chroma.NameVariableGlobal] = fmtName
	h.formats[chroma.NameVariableInstance] = fmtName
	h.formats[chroma.NameVariableMagic] = fmtName

	// Operators
	h.formats[chroma.Operator] = fmtOperator
	h.formats[chroma.OperatorWord] = fmtKeyword

	// Punctuation
	fmtPunct := gui.NewQTextCharFormat()
	fmtPunct.SetForeground(gui.NewQBrush3(gui.NewQColor3(248, 248, 242, 255), 1))
	h.formats[chroma.Punctuation] = fmtPunct

	// Generic
	h.formats[chroma.GenericDeleted] = fmtComment
	h.formats[chroma.GenericEmph] = fmtComment
	h.formats[chroma.GenericError] = fmtOperator
	h.formats[chroma.GenericHeading] = fmtKeyword
	h.formats[chroma.GenericInserted] = fmtString
	h.formats[chroma.GenericOutput] = fmtName
	h.formats[chroma.GenericPrompt] = fmtBuiltin
	h.formats[chroma.GenericStrong] = fmtKeyword
	h.formats[chroma.GenericSubheading] = fmtKeyword
	h.formats[chroma.GenericTraceback] = fmtComment
	h.formats[chroma.GenericUnderline] = fmtName
}

// SetLanguage настраивает лексер на основе имени/расширения файла
func (h *SyntaxHighlighter) SetLanguage(filename string) {
	if filename == "" {
		h.lexer = lexers.Fallback
		h.Rehighlight()
		return
	}

	// Получаем имя файла без пути
	baseName := filepath.Base(filename)
	ext := strings.ToLower(filepath.Ext(filename))

	// Пробуем определить лексер по имени файла
	lexer := lexers.Match(baseName)

	// Если не нашли по имени — пробуем по расширению
	if lexer == nil && ext != "" {
		lexer = lexers.Match("file" + ext)
	}

	// Специальные случаи для Go-экосистемы
	if lexer == nil {
		switch {
		case baseName == "go.mod" || baseName == "go.sum":
			lexer = lexers.Get("go")
		case baseName == "Makefile" || strings.HasSuffix(baseName, ".mk"):
			lexer = lexers.Get("make")
		case baseName == "Dockerfile" || strings.HasSuffix(ext, ".dockerfile"):
			lexer = lexers.Get("docker")
		case ext == ".tmpl":
			// Go templates — используем HTML как базу
			lexer = lexers.Get("html")
		}
	}

	// Fallback на plain text
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Coalesce объединяет соседние токены одного типа — улучшает производительность
	h.lexer = chroma.Coalesce(lexer)

	// Перерисовываем весь документ
	h.Rehighlight()
}

// highlightLine подсвечивает одну строку текста
func (h *SyntaxHighlighter) highlightLine(text string) {
	if h.lexer == nil || text == "" {
		return
	}

	// Токенизируем строку
	iterator, err := h.lexer.Tokenise(nil, text)
	if err != nil {
		return
	}

	// Позиция в строке (в рунах, не в байтах!)
	pos := 0

	for _, token := range iterator.Tokens() {
		tokenText := token.Value
		tokenLen := len([]rune(tokenText)) // Длина в рунах для корректной работы с UTF-8

		// Получаем формат для этого типа токена
		format := h.getFormat(token.Type)

		if format != nil && tokenLen > 0 {
			h.SetFormat(pos, tokenLen, format)
		}

		pos += tokenLen
	}
}

// getFormat возвращает формат для типа токена, учитывая иерархию
func (h *SyntaxHighlighter) getFormat(tokenType chroma.TokenType) *gui.QTextCharFormat {
	// Пробуем найти точное соответствие
	if fmt, ok := h.formats[tokenType]; ok {
		return fmt
	}

	// Пробуем найти по родительскому типу
	// Chroma использует иерархию токенов (например, KeywordDeclaration -> Keyword)
	parent := tokenType.Parent()
	for parent != chroma.None {
		if fmt, ok := h.formats[parent]; ok {
			return fmt
		}
		parent = parent.Parent()
	}

	return nil
}
