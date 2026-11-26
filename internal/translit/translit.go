// Package translit turns a folder name into a short ASCII name for a
// junction: Arabic, Persian and Cyrillic letters by table, Latin accents
// folded, spaces to dashes. Other scripts are dropped; nothing left gives
// "project".
package translit

import "strings"

var table = map[rune]string{
	// Arabic and Persian
	'ا': "a", 'أ': "a", 'إ': "i", 'آ': "a", 'ٱ': "a", 'ب': "b", 'ت': "t", 'ث': "th", 'ج': "j",
	'ح': "h", 'خ': "kh", 'د': "d", 'ذ': "dh", 'ر': "r", 'ز': "z", 'س': "s", 'ش': "sh", 'ص': "s",
	'ض': "d", 'ط': "t", 'ظ': "z", 'ع': "a", 'غ': "gh", 'ف': "f", 'ق': "q", 'ك': "k", 'ل': "l",
	'م': "m", 'ن': "n", 'ه': "h", 'و': "w", 'ي': "y", 'ى': "a", 'ة': "a", 'ئ': "y", 'ؤ': "w",
	'پ': "p", 'چ': "ch", 'ژ': "zh", 'گ': "g", 'ک': "k", 'ی': "y",
	'٠': "0", '١': "1", '٢': "2", '٣': "3", '٤': "4", '٥': "5", '٦': "6", '٧': "7", '٨': "8", '٩': "9",
	'۰': "0", '۱': "1", '۲': "2", '۳': "3", '۴': "4", '۵': "5", '۶': "6", '۷': "7", '۸': "8", '۹': "9",
	// Cyrillic
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo", 'ж': "zh", 'з': "z",
	'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o", 'п': "p", 'р': "r",
	'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch",
	'ы': "y", 'э': "e", 'ю': "yu", 'я': "ya", 'і': "i", 'ї': "yi", 'є': "ye", 'ґ': "g", 'ў': "u",
	// Latin with accents
	'à': "a", 'á': "a", 'â': "a", 'ã': "a", 'ä': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'æ': "ae", 'ç': "c", 'ć': "c", 'č': "c", 'ď': "d", 'đ': "d", 'ð': "d",
	'è': "e", 'é': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ė': "e", 'ę': "e", 'ě': "e",
	'ğ': "g", 'ģ': "g", 'ì': "i", 'í': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i", 'ı': "i",
	'ķ': "k", 'ł': "l", 'ľ': "l", 'ļ': "l", 'ñ': "n", 'ń': "n", 'ň': "n", 'ņ': "n",
	'ò': "o", 'ó': "o", 'ô': "o", 'õ': "o", 'ö': "o", 'ø': "o", 'ō': "o", 'ő': "o", 'œ': "oe",
	'ŕ': "r", 'ř': "r", 'ś': "s", 'š': "s", 'ş': "s", 'ș': "s", 'ß': "ss", 'ť': "t", 'ţ': "t", 'ț': "t",
	'ù': "u", 'ú': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'ý': "y", 'ÿ': "y", 'ź': "z", 'ż': "z", 'ž': "z", 'þ': "th",
}

func Name(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		var out string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '_':
			out = string(r)
		case r == ' ', r == '-', r == '\t':
			out = "-"
		default:
			out = table[r]
		}
		if out == "-" && (b.Len() == 0 || strings.HasSuffix(b.String(), "-")) {
			continue
		}
		b.WriteString(out)
	}
	if name := strings.Trim(b.String(), "-."); name != "" {
		return name
	}
	return "project"
}
