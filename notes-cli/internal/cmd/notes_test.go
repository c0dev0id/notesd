package cmd

import (
	"testing"
)

func TestParseEditorContent(t *testing.T) {
	cases := []struct {
		name        string
		input       string
		wantTitle   string
		wantContent string
	}{
		{
			name:        "title and content",
			input:       "Title: My Note\n---\nHello world",
			wantTitle:   "My Note",
			wantContent: "Hello world",
		},
		{
			name:        "title only no separator",
			input:       "Title: My Note",
			wantTitle:   "My Note",
			wantContent: "",
		},
		{
			name:        "empty title",
			input:       "Title: \n---\nSome content",
			wantTitle:   "",
			wantContent: "Some content",
		},
		{
			name:        "multiline content",
			input:       "Title: Multi\n---\nLine one\nLine two\nLine three",
			wantTitle:   "Multi",
			wantContent: "Line one\nLine two\nLine three",
		},
		{
			name:        "content with separator-like text",
			input:       "Title: Test\n---\nFirst part\n---\nSecond part",
			wantTitle:   "Test",
			wantContent: "First part\n---\nSecond part",
		},
		{
			name:        "title with leading spaces stripped",
			input:       "Title:   Spaced Title   \n---\ncontent",
			wantTitle:   "Spaced Title",
			wantContent: "content",
		},
		{
			name:        "empty content after separator",
			input:       "Title: Empty\n---\n",
			wantTitle:   "Empty",
			wantContent: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			title, content, err := parseEditorContent(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			t.Logf("input=%q title=%q content=%q", tc.input, title, content)
			if title != tc.wantTitle {
				t.Errorf("title: got %q, want %q", title, tc.wantTitle)
			}
			if content != tc.wantContent {
				t.Errorf("content: got %q, want %q", content, tc.wantContent)
			}
		})
	}
}
