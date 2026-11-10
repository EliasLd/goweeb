package metadata

import (
	"testing"
)

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{
			name:  "plain casefold",
			title: "One Piece",
			want:  "one piece",
		},
		{
			name:  "punctuation folds to spaces",
			title: "Fate/Zero: Vol. 2",
			want:  "fate zero vol 2",
		},
		{
			name:  "apostrophes and exclamations vanish",
			title: "Kono Subarashii Sekai ni Shukufuku wo!",
			want:  "kono subarashii sekai ni shukufuku wo",
		},
		{
			name:  "whitespace collapses",
			title: "  Berserk\u00a0 ",
			want:  "berserk",
		},
		{
			name:  "empty",
			title: "  ",
			want:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizeTitle(test.title); got != test.want {
				t.Errorf(
					"NormalizeTitle(%q) = %q, want %q",
					test.title,
					got,
					test.want,
				)
			}
		})
	}
}

func TestLanguageISO(t *testing.T) {
	tests := []struct {
		scanPath string
		want     string
	}{
		{scanPath: "en", want: "en"},
		{scanPath: "pt-br", want: "pt-BR"},
		{scanPath: "zh-hk", want: "zh-HK"},
		{scanPath: "ja", want: "ja"},
		{scanPath: "pt-br ", want: "pt-BR"},
		{scanPath: "zh-hans", want: "zh-hans"},
		{scanPath: "", want: ""},
		{scanPath: "English", want: ""},
		{scanPath: "e", want: ""},
		{scanPath: "-br", want: ""},
	}

	for _, test := range tests {
		if got := LanguageISO(test.scanPath); got != test.want {
			t.Errorf(
				"LanguageISO(%q) = %q, want %q",
				test.scanPath,
				got,
				test.want,
			)
		}
	}
}

func TestCoverExtension(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{url: "https://cdn.example.com/c/x.jpg", want: ".jpg"},
		{url: "https://cdn.example.com/c/x.jpeg?v=1", want: ".jpeg"},
		{url: "https://cdn.example.com/c/x.png", want: ".png"},
		{url: "https://cdn.example.com/c/x.webp", want: ".webp"},
		{url: "https://cdn.example.com/c/x?download", want: ".jpg"},
		{url: "https://cdn.example.com/c/x", want: ".jpg"},
		{url: "not-a-url", want: ".jpg"},
	}

	for _, test := range tests {
		if got := CoverExtension(test.url); got != test.want {
			t.Errorf(
				"CoverExtension(%q) = %q, want %q",
				test.url,
				got,
				test.want,
			)
		}
	}
}

func TestSanitizeDescription(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "paragraph breaks stay",
			body: "First line<br><br>Second line(Source: Somebody)",
			want: "First line\nSecond line(Source: Somebody)",
		},
		{
			name: "variant br tags",
			body: "One<br />\nTwo<br/>Three",
			want: "One\nTwo\nThree",
		},
		{
			name: "double encoded entities",
			body: "A &#039;quoted&#039; name and &amp;amp;b",
			want: "A 'quoted' name and &b",
		},
		{
			name: "other tags stripped",
			body: "<b>Bold</b> plain",
			want: "Bold plain",
		},
		{
			name: "blank input",
			body: "   ",
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sanitizeDescription(test.body)

			if got != test.want {
				t.Errorf(
					"sanitizeDescription(%q) = %q, want %q",
					test.body,
					got,
					test.want,
				)
			}
		})
	}
}
