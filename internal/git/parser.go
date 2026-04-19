package git

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"mood-ring/internal/types"
)

// revertSubjectRE matches the common Revert commit subject forms.
var revertSubjectRE = regexp.MustCompile(`^(?:Revert\s+["']|revert\s*[:(])`)

// revertTargetRE extracts the short hash from Revert '... (abcdef0)' subjects.
var revertTargetRE = regexp.MustCompile(`\(([0-9a-f]{7,})\)`)

// ParseGitLog turns raw `git log --numstat --format=GitFormat` output into
// CommitRecords. Splits on RecordSep, parses header line (6 fields), then
// accumulates numstat lines until the next record.
func ParseGitLog(raw string) ([]types.CommitRecord, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	records := strings.Split(raw, RecordSep)
	var out []types.CommitRecord
	for _, rec := range records {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		commit, ok := parseRecord(rec)
		if ok {
			out = append(out, commit)
		}
	}
	return out, nil
}

func parseRecord(rec string) (types.CommitRecord, bool) {
	lines := strings.Split(strings.TrimLeft(rec, "\n"), "\n")
	if len(lines) == 0 {
		return types.CommitRecord{}, false
	}
	header := lines[0]
	fields := strings.Split(header, FieldSep)
	if len(fields) < 6 {
		return types.CommitRecord{}, false
	}

	committerDate, err := time.Parse(time.RFC3339, fields[1])
	if err != nil {
		return types.CommitRecord{}, false
	}
	authorDate, err := time.Parse(time.RFC3339, fields[2])
	if err != nil {
		authorDate = committerDate
	}

	message := strings.Join(fields[5:], FieldSep) // safety if FIELD_SEP appeared in subject
	isRevert := revertSubjectRE.MatchString(message)
	revertTarget := ""
	if isRevert {
		if m := revertTargetRE.FindStringSubmatch(message); m != nil {
			revertTarget = m[1]
		}
	}

	files := make([]types.FileChange, 0)
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		added := atoiOrZero(parts[0])
		removed := atoiOrZero(parts[1])
		path := parts[2]
		files = append(files, types.FileChange{Path: path, Added: added, Removed: removed})
	}

	return types.CommitRecord{
		Hash:              fields[0],
		Date:              committerDate,
		AuthorDate:        authorDate,
		AuthorName:        fields[3],
		AuthorEmail:       fields[4],
		Message:           message,
		FilesChanged:      files,
		IsRevert:          isRevert,
		RevertedHashShort: revertTarget,
	}, true
}

func atoiOrZero(s string) int {
	s = strings.TrimSpace(s)
	if s == "-" || s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
