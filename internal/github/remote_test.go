package github

import "testing"

func TestParseRemoteURL_SSHForm(t *testing.T) {
	cases := []struct {
		in            string
		wantOwner     string
		wantRepo      string
	}{
		{"git@github.com:anthropic/claude-code.git", "anthropic", "claude-code"},
		{"git@github.com:anthropic/claude-code", "anthropic", "claude-code"},
		{"git@github.com:org-name/repo-name.git", "org-name", "repo-name"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, ok := ParseRemoteURL(c.in)
			if !ok {
				t.Fatalf("ParseRemoteURL(%q) returned ok=false", c.in)
			}
			if got.Owner != c.wantOwner || got.Name != c.wantRepo {
				t.Errorf("got %s/%s, want %s/%s", got.Owner, got.Name, c.wantOwner, c.wantRepo)
			}
		})
	}
}

func TestParseRemoteURL_HTTPSForm(t *testing.T) {
	cases := []string{
		"https://github.com/anthropic/claude-code.git",
		"https://github.com/anthropic/claude-code",
		"https://github.com/anthropic/claude-code/",
		"http://github.com/anthropic/claude-code",
		"https://user:token@github.com/anthropic/claude-code.git",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, ok := ParseRemoteURL(in)
			if !ok {
				t.Fatalf("ParseRemoteURL(%q) returned ok=false", in)
			}
			if got.Owner != "anthropic" || got.Name != "claude-code" {
				t.Errorf("got %s/%s, want anthropic/claude-code", got.Owner, got.Name)
			}
		})
	}
}

func TestParseRemoteURL_RejectsNonGitHub(t *testing.T) {
	cases := []string{
		"git@gitlab.com:foo/bar.git",
		"https://bitbucket.org/foo/bar",
		"/local/path/to/repo",
		"",
		"not a url",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, ok := ParseRemoteURL(in); ok {
				t.Errorf("ParseRemoteURL(%q) accepted a non-GitHub remote", in)
			}
		})
	}
}

func TestOwnerRepoString(t *testing.T) {
	r := OwnerRepo{Owner: "foo", Name: "bar"}
	if r.String() != "foo/bar" {
		t.Errorf("got %s", r.String())
	}
}
