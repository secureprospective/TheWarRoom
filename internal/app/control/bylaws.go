package control

import (
	"bufio"
	"bytes"
	"strings"
)

type BylawSection struct {
	Title string   `json:"title"`
	Lines []string `json:"lines"`
}

// BylawSections reads the rulebook's markdown: each "## " heading starts a section; "### "
// headings, bullets, paragraphs and table rows become plain lines; markup is dropped.
func BylawSections(md []byte) []BylawSection {
	var out []BylawSection
	sc := bufio.NewScanner(bytes.NewReader(md))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "## "):
			out = append(out, BylawSection{Title: plainMarkdown(strings.TrimPrefix(line, "## "))})
		case len(out) == 0 || line == "" || line == "---" || isTableRule(line):
			continue
		default:
			if text := bylawLine(line); text != "" {
				out[len(out)-1].Lines = append(out[len(out)-1].Lines, text)
			}
		}
	}
	return out
}

func bylawLine(line string) string {
	switch {
	case strings.HasPrefix(line, "### "):
		return plainMarkdown(strings.TrimPrefix(line, "### ")) + ":"
	case strings.HasPrefix(line, "|"):
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			if c = plainMarkdown(c); c != "" {
				cells = append(cells, c)
			}
		}
		return strings.Join(cells, " · ")
	case strings.HasPrefix(line, "- "):
		return "• " + plainMarkdown(strings.TrimPrefix(line, "- "))
	}
	return plainMarkdown(line)
}

func isTableRule(line string) bool {
	return strings.HasPrefix(line, "|") && strings.Trim(line, "|-: ") == ""
}

func plainMarkdown(s string) string {
	return strings.TrimSpace(strings.NewReplacer("**", "", "`", "").Replace(s))
}
