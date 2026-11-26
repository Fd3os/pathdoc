package translit

import "testing"

func TestName(t *testing.T) {
	cases := map[string]string{
		"ابوي":       "abwy",
		"متجر":       "mtjr",
		"Мой проект": "moy-proekt",
		"José's App": "joses-app",
		"Straße":     "strasse",
		"café":       "cafe",
		"张伟的项目":      "project",
		"My Apps":    "my-apps",
		"  a  b  ":   "a-b",
		"گل پیک":     "gl-pyk",
		"v1.2_beta":  "v1.2_beta",
		"":           "project",
	}
	for in, want := range cases {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
}
