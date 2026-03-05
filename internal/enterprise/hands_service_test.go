package enterprise

import "testing"

func TestParseHandMarkdown(t *testing.T) {
	md := "---\nname: daily-report\ndescription: A daily summary\n---\nGenerate the report."
	name, desc, content := parseHandMarkdown(md)
	if name != "daily-report" {
		t.Fatalf("name = %q, want daily-report", name)
	}
	if desc != "A daily summary" {
		t.Fatalf("desc = %q, want A daily summary", desc)
	}
	if content != "Generate the report." {
		t.Fatalf("content = %q", content)
	}
}

func TestParseHandMarkdownNoFrontmatter(t *testing.T) {
	md := "Just some plain content."
	name, desc, content := parseHandMarkdown(md)
	if name != "unnamed" {
		t.Fatalf("name = %q, want unnamed", name)
	}
	if desc != "" {
		t.Fatalf("desc = %q, want empty", desc)
	}
	if content != "Just some plain content." {
		t.Fatalf("content = %q", content)
	}
}

func TestParseHandMarkdownEmptyName(t *testing.T) {
	md := "---\ndescription: test\n---\nContent here."
	name, _, _ := parseHandMarkdown(md)
	if name != "unnamed" {
		t.Fatalf("name = %q, want unnamed", name)
	}
}
