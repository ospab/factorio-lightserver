package main

var russian = systemRussian()

// setLang picks the language of the messages: "en", "ru" or "auto".
func setLang(lang string) {
	switch lang {
	case "en":
		russian = false
	case "ru":
		russian = true
	}
}

func tr(en, ru string) string {
	if russian {
		return ru
	}
	return en
}

// systemRussian reports whether the Windows display language is Russian.
func systemRussian() bool {
	const langRussian = 0x19
	id, _, _ := getUserDefaultUILanguage.Call()
	return id&0x3ff == langRussian
}
