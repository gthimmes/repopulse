package codeowners

import (
	"reflect"
	"testing"
)

func TestParse_BasicLines(t *testing.T) {
	co := Parse("*.js @frontend\n*.ts @backend")
	if len(co.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(co.Rules))
	}
	if !reflect.DeepEqual(co.Rules[0].Owners, []string{"@frontend"}) {
		t.Errorf("rule 0 owners: %v", co.Rules[0].Owners)
	}
	if !reflect.DeepEqual(co.Rules[1].Owners, []string{"@backend"}) {
		t.Errorf("rule 1 owners: %v", co.Rules[1].Owners)
	}
}

func TestParse_CommentsAndBlankLines(t *testing.T) {
	co := Parse(`
# Global comment
*.js  @frontend   # trailing comment

# another block
*.ts  @backend
`)
	if len(co.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(co.Rules))
	}
}

func TestParse_MultipleOwners(t *testing.T) {
	co := Parse("src/payments/ @org/payments-team @alice @bob")
	want := []string{"@org/payments-team", "@alice", "@bob"}
	if !reflect.DeepEqual(co.Rules[0].Owners, want) {
		t.Errorf("owners: %v", co.Rules[0].Owners)
	}
}

func TestOwnersFor_UnanchoredPattern(t *testing.T) {
	co := Parse("*.md @docs")
	if got := co.OwnersFor("README.md"); !reflect.DeepEqual(got, []string{"@docs"}) {
		t.Errorf("README.md: %v", got)
	}
	if got := co.OwnersFor("src/deep/nested/notes.md"); !reflect.DeepEqual(got, []string{"@docs"}) {
		t.Errorf("nested: %v", got)
	}
	if got := co.OwnersFor("src/foo.ts"); got != nil && len(got) != 0 {
		t.Errorf("non-match should be empty, got %v", got)
	}
}

func TestOwnersFor_AnchoredPattern(t *testing.T) {
	co := Parse("/src/payments/ @payments")
	if got := co.OwnersFor("src/payments/ledger.ts"); !reflect.DeepEqual(got, []string{"@payments"}) {
		t.Errorf("want @payments, got %v", got)
	}
	if got := co.OwnersFor("src/payments/deep/file.ts"); !reflect.DeepEqual(got, []string{"@payments"}) {
		t.Errorf("deep: want @payments, got %v", got)
	}
	if got := co.OwnersFor("vendor/src/payments/foo.ts"); len(got) != 0 {
		t.Errorf("vendor shouldn't match anchored, got %v", got)
	}
}

func TestOwnersFor_DirectoryPattern(t *testing.T) {
	co := Parse("docs/ @docs-team")
	if got := co.OwnersFor("docs/README.md"); !reflect.DeepEqual(got, []string{"@docs-team"}) {
		t.Errorf("docs/README: %v", got)
	}
	if got := co.OwnersFor("docs/a/b/c.md"); !reflect.DeepEqual(got, []string{"@docs-team"}) {
		t.Errorf("nested: %v", got)
	}
	if got := co.OwnersFor("docs.md"); len(got) != 0 {
		t.Errorf("docs.md should not match: %v", got)
	}
}

func TestOwnersFor_LastMatchWins(t *testing.T) {
	co := Parse(`
*  @default
/src/  @engineers
/src/payments/  @payments
`)
	cases := []struct {
		path string
		want []string
	}{
		{"README.md", []string{"@default"}},
		{"src/auth.ts", []string{"@engineers"}},
		{"src/payments/ledger.ts", []string{"@payments"}},
	}
	for _, c := range cases {
		if got := co.OwnersFor(c.path); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: want %v, got %v", c.path, c.want, got)
		}
	}
}

func TestOwnersFor_WindowsPaths(t *testing.T) {
	co := Parse("/src/payments/ @payments")
	if got := co.OwnersFor(`src\payments\ledger.ts`); !reflect.DeepEqual(got, []string{"@payments"}) {
		t.Errorf("got %v", got)
	}
}

func TestOwnersFor_NoMatch(t *testing.T) {
	co := Parse("*.md @docs")
	if got := co.OwnersFor("src/unowned.ts"); len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

func TestParse_SkipLinesWithoutOwners(t *testing.T) {
	co := Parse(`
*.orphan
*.md @docs
`)
	if len(co.Rules) != 1 {
		t.Errorf("want 1 rule, got %d", len(co.Rules))
	}
}

func TestAggregate_NilCodeowners(t *testing.T) {
	got := AggregateOwnersForModule([]string{"src/payments/ledger.ts"}, nil, 2)
	if len(got) != 0 {
		t.Errorf("nil codeowners should yield empty, got %v", got)
	}
}

func TestAggregate_RanksByFileCount(t *testing.T) {
	co := Parse(`
/src/payments/  @payments
/src/auth/      @security @backend
*.md            @docs
`)
	files := []string{
		"src/payments/ledger.ts",
		"src/payments/fees.ts",
		"src/payments/tax.ts",
		"src/auth/session.ts",
	}
	got := AggregateOwnersForModule(files, co, 2)
	if len(got) != 2 {
		t.Fatalf("want 2 owners, got %d (%v)", len(got), got)
	}
	if got[0] != "@payments" {
		t.Errorf("top should be @payments, got %s", got[0])
	}
}

func TestAggregate_SkipsUnowned(t *testing.T) {
	co := Parse(`
/src/payments/  @payments
*.md            @docs
`)
	files := []string{"src/unowned.ts", "src/payments/ledger.ts"}
	got := AggregateOwnersForModule(files, co, 2)
	if !reflect.DeepEqual(got, []string{"@payments"}) {
		t.Errorf("want [@payments], got %v", got)
	}
}
