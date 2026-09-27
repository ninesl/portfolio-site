package pages

// SettingsChoice defines the site's text font, code font, and code theme.
type SettingsChoice struct {
	TextFont    string
	TextFontURL string
	CodeFont    string
	CodeFontURL string
	SyntaxTheme string
}

var DefaultSettings = SettingsChoice{
	TextFont:    "Varela Round",
	TextFontURL: "/assets/fonts/VarelaRound-Regular.ttf",
	CodeFont:    "Code New Roman",
	CodeFontURL: "/assets/fonts/Code%20New%20Roman.otf",
	SyntaxTheme: "github-dark",
}
