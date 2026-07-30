package casebook

import (
	"embed"
	"fmt"
	"io"
	"strings"
)

//go:embed cases/*.md
var caseFiles embed.FS

type Entry struct {
	ID    string
	Title string
	File  string
}

var catalog = []Entry{
	{
		ID:    "CND-CASE-2026-001",
		Title: "v2rayN TUN 因 Windows 排除端口范围无法启动",
		File:  "cases/CND-CASE-2026-001.md",
	},
}

func List() []Entry {
	return append([]Entry(nil), catalog...)
}

func Lookup(id string) (string, bool) {
	for _, entry := range catalog {
		if !strings.EqualFold(strings.TrimSpace(id), entry.ID) {
			continue
		}
		content, err := caseFiles.ReadFile(entry.File)
		if err != nil {
			return "", false
		}
		return string(content), true
	}
	return "", false
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && strings.EqualFold(args[0], "cases") {
		for _, entry := range List() {
			_, _ = fmt.Fprintf(stdout, "%s\t%s\n", entry.ID, entry.Title)
		}
		return 0
	}
	if len(args) == 2 && strings.EqualFold(args[0], "case") {
		content, ok := Lookup(args[1])
		if !ok {
			_, _ = fmt.Fprintf(stderr, "unknown case %q\n", args[1])
			return 1
		}
		_, _ = io.WriteString(stdout, content)
		if !strings.HasSuffix(content, "\n") {
			_, _ = io.WriteString(stdout, "\n")
		}
		return 0
	}
	_, _ = fmt.Fprintln(stderr, "usage: cnd cases | cnd case <case-id>")
	return 2
}
