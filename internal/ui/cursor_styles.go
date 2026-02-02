package ui

// CursorStyle описывает стиль курсора
type CursorStyle struct {
	Name        string
	Width       int
	Description string
}

// CursorStyles — доступные стили курсора
var CursorStyles = map[string]*CursorStyle{
	"Line": {
		Name:        "Line",
		Width:       2,
		Description: "Thin line cursor (default)",
	},
	"Thin Line": {
		Name:        "Thin Line",
		Width:       1,
		Description: "Very thin line cursor",
	},
	"Thick Line": {
		Name:        "Thick Line",
		Width:       4,
		Description: "Thick line cursor",
	},
	"Block": {
		Name:        "Block",
		Width:       8,
		Description: "Block cursor (like Vim normal mode)",
	},
	"Wide Block": {
		Name:        "Wide Block",
		Width:       12,
		Description: "Wide block cursor",
	},
	"Underline": {
		Name:        "Underline",
		Width:       6,
		Description: "Medium width (underline simulation)",
	},
}

// CursorStyleOrder — порядок отображения в меню
var CursorStyleOrder = []string{
	"Thin Line",
	"Line",
	"Thick Line",
	"Block",
	"Wide Block",
	"Underline",
}
