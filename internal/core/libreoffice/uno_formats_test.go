package libreoffice

import "testing"

func TestIsUnoSupportedFormat(t *testing.T) {
	cases := []struct {
		name   string
		format string
		want   bool
	}{
		{name: "pdf", format: "pdf", want: true},
		{name: "docx", format: "docx", want: true},
		{name: "xlsx", format: "xlsx", want: true},
		{name: "pptx", format: "pptx", want: true},
		{name: "unknown", format: "odt", want: false},
		{name: "uppercase", format: "PDF", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUnoSupportedFormat(tc.format); got != tc.want {
				t.Fatalf("期望 %v, 实际 %v", tc.want, got)
			}
		})
	}
}
