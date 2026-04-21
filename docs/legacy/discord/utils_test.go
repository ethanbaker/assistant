package main

import "testing"

func TestSanitizeHTMLtoDiscordMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, <STRONG>world</STRONG>!", "Hello, **world**!"},
		{"This is <EM>italic</EM> text.", "This is *italic* text."},
		{"<U>Underline</U> this.", "__Underline__ this."},
		{"This is <S>strikethrough</S> text.", "This is ~~strikethrough~~ text."},
		{"Here is some <CODE>inline code</CODE>.", "Here is some `inline code`."},
		{"<PRE><CODE class=\"language-go\">fmt.Println(\"Hello, World!\")</CODE></PRE>", "```go\nfmt.Println(\"Hello, World!\")\n```"},
		{"<PRE><CODE>Some code block</CODE></PRE>", "```\nSome code block\n```"},
		{"Mixed <B>bold</B> and <I>italic</I> text.", "Mixed **bold** and *italic* text."},
		{"Nested <B>bold and <I>italic</I></B> text.", "Nested **bold and *italic*** text."},
		{"No HTML here!", "No HTML here!"},
		{"<DIV>Some <B>bold</B> text in a div.</DIV>", "Some **bold** text in a div."},
	}

	for _, test := range tests {
		result := sanitizeHTMLToDiscordMarkdown(test.input)
		if result != test.expected {
			t.Errorf("For input '%s', expected '%s' but got '%s'", test.input, test.expected, result)
		}
	}
}
