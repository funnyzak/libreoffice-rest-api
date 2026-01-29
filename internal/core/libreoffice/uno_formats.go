package libreoffice

import "strings"

var unoSupportedFormats = map[string]struct{}{
	"pdf":  {},
	"docx": {},
	"xlsx": {},
	"pptx": {},
}

// IsUnoSupportedFormat 判断 UNO 是否支持输出格式。
func IsUnoSupportedFormat(format string) bool {
	format = strings.ToLower(strings.TrimSpace(format))
	_, ok := unoSupportedFormats[format]
	return ok
}
