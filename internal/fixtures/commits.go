// Package fixtures provides deterministic commit fixtures for tests.
// Port of src/__fixtures__/sampleCommits.ts.
package fixtures

import (
	"regexp"
	"time"

	"mood-ring/internal/types"
)

var revertRE = regexp.MustCompile(`^(?:Revert\s+["']|revert\s*[:(])`)

type fileSpec struct {
	Path    string
	Added   int
	Removed int
}

type authorSpec struct {
	Name  string
	Email string
	Hour  int
}

func commit(hash string, daysAgo int, message string, files []fileSpec, a ...authorSpec) types.CommitRecord {
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location()).AddDate(0, 0, -daysAgo)
	author := authorSpec{Name: "Dev One", Email: "dev.one@example.com", Hour: 10}
	if len(a) > 0 {
		if a[0].Name != "" {
			author.Name = a[0].Name
		}
		if a[0].Email != "" {
			author.Email = a[0].Email
		}
		if a[0].Hour != 0 {
			author.Hour = a[0].Hour
			date = time.Date(date.Year(), date.Month(), date.Day(), author.Hour, 0, 0, 0, date.Location())
		}
	}
	fc := make([]types.FileChange, len(files))
	for i, f := range files {
		fc[i] = types.FileChange{Path: f.Path, Added: f.Added, Removed: f.Removed}
	}
	return types.CommitRecord{
		Hash:         hash,
		Date:         date,
		AuthorDate:   date,
		AuthorName:   author.Name,
		AuthorEmail:  author.Email,
		Message:      message,
		FilesChanged: fc,
		IsRevert:     revertRE.MatchString(message),
	}
}

// CalmFixture — ~60 commits spread evenly, no bug matches, low churn.
func CalmFixture() []types.CommitRecord {
	f := func(path string, a, r int) []fileSpec { return []fileSpec{{path, a, r}} }
	return []types.CommitRecord{
		commit("a001", 89, "feat: initialize project structure", f("src/index.ts", 25, 0)),
		commit("a002", 87, "feat: add user model", f("src/models/user.ts", 30, 0)),
		commit("a003", 85, "feat: add database connection module", f("src/db/connection.ts", 40, 0)),
		commit("a004", 83, "test: add user model tests", f("src/models/__tests__/user.test.ts", 35, 0)),
		commit("a005", 81, "refactor: extract validation helpers", f("src/utils/validate.ts", 20, 5)),
		commit("a006", 80, "chore: update eslint config", f("eslintrc.json", 10, 8)),
		commit("a007", 78, "feat: add authentication middleware", f("src/middleware/auth.ts", 45, 0)),
		commit("a008", 76, "docs: add contributing guide", f("CONTRIBUTING.md", 30, 0)),
		commit("a009", 74, "feat: add login endpoint", f("src/routes/login.ts", 35, 0)),
		commit("a010", 73, "test: add auth middleware tests", f("src/middleware/__tests__/auth.test.ts", 40, 0)),
		commit("a011", 71, "feat: add user profile page", f("src/pages/Profile.tsx", 45, 0)),
		commit("a012", 69, "refactor: simplify db queries", f("src/db/queries.ts", 15, 20)),
		commit("a013", 67, "feat: add logout functionality", f("src/routes/logout.ts", 20, 0)),
		commit("a014", 66, "chore: add prettier config", f("prettier.config.js", 8, 0)),
		commit("a015", 64, "feat: add settings page", f("src/pages/Settings.tsx", 40, 0)),
		commit("a016", 62, "test: add login endpoint tests", f("src/routes/__tests__/login.test.ts", 30, 0)),
		commit("a017", 60, "feat: add email notifications", f("src/services/email.ts", 35, 0)),
		commit("a018", 59, "refactor: rename config keys", f("src/config.ts", 10, 10)),
		commit("a019", 57, "feat: add password reset flow", f("src/routes/reset.ts", 30, 0)),
		commit("a020", 55, "docs: update readme with setup steps", f("README.md", 25, 5)),
		commit("a021", 53, "feat: add role-based access control", f("src/middleware/rbac.ts", 40, 0)),
		commit("a022", 52, "test: add rbac tests", f("src/middleware/__tests__/rbac.test.ts", 35, 0)),
		commit("a023", 50, "feat: add user search endpoint", f("src/routes/search.ts", 25, 0)),
		commit("a024", 48, "refactor: consolidate route handlers", f("src/routes/index.ts", 15, 10)),
		commit("a025", 47, "feat: add pagination helpers", f("src/utils/pagination.ts", 20, 0)),
		commit("a026", 45, "chore: update typescript version", f("package.json", 2, 2)),
		commit("a027", 43, "feat: add activity log", f("src/services/activityLog.ts", 30, 0)),
		commit("a028", 42, "test: add email service tests", f("src/services/__tests__/email.test.ts", 25, 0)),
		commit("a029", 40, "feat: add dashboard overview", f("src/pages/Dashboard.tsx", 45, 0)),
		commit("a030", 38, "refactor: extract date formatters", f("src/utils/dates.ts", 15, 0)),
		commit("a031", 37, "feat: add user avatar upload", f("src/routes/avatar.ts", 30, 0)),
		commit("a032", 35, "docs: document API endpoints", f("docs/api.md", 40, 5)),
		commit("a033", 33, "feat: add notification preferences", f("src/pages/Notifications.tsx", 35, 0)),
		commit("a034", 32, "test: add dashboard tests", f("src/pages/__tests__/Dashboard.test.tsx", 30, 0)),
		commit("a035", 30, "feat: add export to CSV", f("src/services/export.ts", 25, 0)),
		commit("a036", 28, "refactor: improve error messages", f("src/utils/errors.ts", 10, 8)),
		commit("a037", 27, "feat: add team management", f("src/routes/teams.ts", 35, 0)),
		commit("a038", 25, "chore: add CI workflow", f("github/workflows/ci.yml", 20, 0)),
		commit("a039", 23, "feat: add invite flow", f("src/routes/invite.ts", 30, 0)),
		commit("a040", 22, "test: add team management tests", f("src/routes/__tests__/teams.test.ts", 25, 0)),
		commit("a041", 20, "feat: add webhook support", f("src/services/webhooks.ts", 40, 0)),
		commit("a042", 19, "refactor: centralize constants", f("src/constants.ts", 15, 3)),
		commit("a043", 17, "feat: add rate limiting", f("src/middleware/rateLimit.ts", 25, 0)),
		commit("a044", 16, "docs: add architecture diagram", f("docs/architecture.md", 30, 0)),
		commit("a045", 14, "feat: add health check endpoint", f("src/routes/health.ts", 15, 0)),
		commit("a046", 13, "test: add webhook tests", f("src/services/__tests__/webhooks.test.ts", 30, 0)),
		commit("a047", 11, "feat: add graceful shutdown", f("src/server.ts", 20, 5)),
		commit("a048", 10, "refactor: improve type safety", f("src/types.ts", 15, 10)),
		commit("a049", 9, "feat: add request logging", f("src/middleware/logger.ts", 25, 0)),
		commit("a050", 8, "chore: update dependencies", f("package.json", 5, 5)),
		commit("a051", 7, "feat: add response compression", f("src/middleware/compress.ts", 15, 0)),
		commit("a052", 6, "test: add rate limit tests", f("src/middleware/__tests__/rateLimit.test.ts", 20, 0)),
		commit("a053", 5, "feat: add cors configuration", f("src/middleware/cors.ts", 10, 0)),
		commit("a054", 5, "docs: add deployment guide", f("docs/deploy.md", 35, 0)),
		commit("a055", 4, "refactor: clean up imports", f("src/routes/index.ts", 8, 12)),
		commit("a056", 3, "feat: add metrics endpoint", f("src/routes/metrics.ts", 20, 0)),
		commit("a057", 3, "test: add integration tests", f("src/__tests__/integration.test.ts", 40, 0)),
		commit("a058", 2, "chore: add pre-commit hooks", f(".husky/pre-commit", 5, 0)),
		commit("a059", 1, "feat: add openapi spec generation", f("src/utils/openapi.ts", 30, 0)),
		commit("a060", 1, "docs: update changelog", f("CHANGELOG.md", 15, 0)),
	}
}

// ChaoticFixture — ~70 commits with bursts and gaps, lots of fixes/reverts.
func ChaoticFixture() []types.CommitRecord {
	f1 := func(path string, a, r int) []fileSpec { return []fileSpec{{path, a, r}} }
	f2 := func(p1 string, a1, r1 int, p2 string, a2, r2 int) []fileSpec {
		return []fileSpec{{p1, a1, r1}, {p2, a2, r2}}
	}
	return []types.CommitRecord{
		commit("z001", 88, "fix: auth completely broken after deploy", f1("src/auth/index.ts", 85, 120)),
		commit("z002", 88, "hotfix: revert last commit", f1("src/auth/index.ts", 120, 85)),
		commit("z003", 87, "fix: token validation regression", f1("src/auth/token.ts", 45, 60)),
		commit("z004", 87, "fix: still broken on prod", f1("src/auth/token.ts", 20, 45)),
		commit("z005", 87, "fix: finally resolved token bug", f1("src/auth/token.ts", 30, 20)),
		commit("z006", 86, "fix: session expiry not working", f1("src/auth/session.ts", 50, 35)),
		commit("z007", 86, "fix: login page crashes on empty form", f1("src/pages/Login.tsx", 25, 40)),
		commit("z008", 85, "feat: add error boundary", f1("src/components/ErrorBoundary.tsx", 60, 0)),
		commit("z009", 85, "hotfix: error boundary breaks routing", f1("src/components/ErrorBoundary.tsx", 40, 30)),
		commit("z010", 85, "revert: remove error boundary for now", f1("src/components/ErrorBoundary.tsx", 0, 70)),
		commit("z011", 84, "fix: broken database migrations", f1("src/db/migrations/001.sql", 30, 20)),
		commit("z012", 84, "fix: migration order wrong", f1("src/db/migrations/001.sql", 15, 30)),
		commit("z013", 84, "feat: add retry logic for db connection", f1("src/db/connection.ts", 40, 10)),
		commit("z014", 83, "fix: retry causes infinite loop", f1("src/db/connection.ts", 20, 25)),
		commit("z015", 83, "fix: connection pool exhaustion", f1("src/db/connection.ts", 35, 20)),
		commit("z016", 82, "feat: add monitoring dashboard", f1("src/pages/Monitor.tsx", 150, 0)),
		commit("z017", 82, "bug: dashboard shows wrong metrics", f1("src/pages/Monitor.tsx", 40, 60)),
		commit("z018", 82, "fix: metrics calculation off by one", f1("src/services/metrics.ts", 15, 10)),
		commit("z019", 81, "feat: add user notification system", f1("src/services/notify.ts", 80, 0)),
		commit("z020", 81, "fix: notifications sent to wrong users", f1("src/services/notify.ts", 30, 25)),
		commit("z021", 80, "hotfix: critical data leak in API response", f2("src/routes/api.ts", 10, 5, "src/middleware/sanitize.ts", 40, 0)),
		commit("z022", 80, "fix: sanitizer too aggressive, strips valid data", f1("src/middleware/sanitize.ts", 25, 30)),
		commit("z023", 79, "feat: add caching layer", f1("src/cache/redis.ts", 70, 0)),
		commit("z024", 79, "fix: cache invalidation broken", f1("src/cache/redis.ts", 30, 20)),
		commit("z025", 79, "fix: cache causes stale reads", f1("src/cache/redis.ts", 25, 30)),
		commit("z026", 78, "revert: remove caching for now", f1("src/cache/redis.ts", 0, 75)),
		commit("z027", 78, "feat: add file upload endpoint", f1("src/routes/upload.ts", 60, 0)),
		commit("z028", 77, "fix: upload fails for large files", f1("src/routes/upload.ts", 20, 15)),
		commit("z029", 77, "fix: upload corrupts binary files", f1("src/routes/upload.ts", 15, 20)),
		commit("z030", 77, "patch: workaround for upload memory leak", f1("src/routes/upload.ts", 10, 5)),
		commit("z031", 76, "feat: add search functionality", f1("src/services/search.ts", 90, 0)),
		commit("z032", 76, "fix: search returns duplicates", f1("src/services/search.ts", 20, 15)),
		commit("z033", 76, "fix: search crashes on special characters", f1("src/services/search.ts", 10, 5)),
		commit("z034", 75, "feat: add admin panel", f1("src/pages/Admin.tsx", 120, 0)),
		commit("z035", 75, "fix: admin panel accessible to non-admins", f2("src/pages/Admin.tsx", 15, 5, "src/middleware/auth.ts", 10, 3)),
		commit("z036", 75, "urgent: disable admin panel in production", f1("src/pages/Admin.tsx", 5, 10)),
		commit("z037", 75, "feat: add database backup script", f1("scripts/backup.sh", 40, 0)),
		commit("z038", 75, "fix: backup script corrupts data", f1("scripts/backup.sh", 20, 15)),
		commit("z039", 75, "regression: backup breaks restore flow", f1("scripts/backup.sh", 15, 20)),
		commit("z040", 75, "feat: add API rate limiting", f1("src/middleware/rateLimit.ts", 35, 0)),
		// 22-day gap
		commit("z041", 7, "feat: new dashboard (rushed)", f1("src/dashboard/index.tsx", 200, 0)),
		commit("z042", 7, "bug: dashboard crashes on load", f1("src/dashboard/index.tsx", 40, 80)),
		commit("z043", 7, "fix: patch dashboard crash", f1("src/dashboard/index.tsx", 15, 40)),
		commit("z044", 6, "feat: add chart component", f1("src/dashboard/Chart.tsx", 100, 0)),
		commit("z045", 6, "fix: chart renders blank", f1("src/dashboard/Chart.tsx", 30, 25)),
		commit("z046", 6, "fix: chart axes wrong", f1("src/dashboard/Chart.tsx", 20, 20)),
		commit("z047", 5, "feat: add data export", f1("src/dashboard/export.ts", 60, 0)),
		commit("z048", 5, "fix: export produces invalid CSV", f1("src/dashboard/export.ts", 25, 20)),
		commit("z049", 5, "hotfix: export crashes on empty dataset", f1("src/dashboard/export.ts", 10, 5)),
		commit("z050", 5, "fix: export missing headers", f1("src/dashboard/export.ts", 8, 3)),
		commit("z051", 4, "feat: add filter controls", f1("src/dashboard/Filters.tsx", 80, 0)),
		commit("z052", 4, "fix: filters don't apply", f1("src/dashboard/Filters.tsx", 20, 15)),
		commit("z053", 4, "fix: filter state resets on navigate", f1("src/dashboard/Filters.tsx", 15, 10)),
		commit("z054", 3, "feat: add user preferences API", f1("src/routes/preferences.ts", 45, 0)),
		commit("z055", 3, "broken: preferences overwrite each other", f1("src/routes/preferences.ts", 20, 15)),
		commit("z056", 3, "fix: race condition in preferences save", f1("src/routes/preferences.ts", 25, 10)),
		commit("z057", 2, "feat: add tooltip component", f1("src/components/Tooltip.tsx", 35, 0)),
		commit("z058", 2, "fix: tooltip flickers on hover", f1("src/components/Tooltip.tsx", 10, 8)),
		commit("z059", 2, "oops: committed debug logs", f1("src/dashboard/index.tsx", 0, 15)),
		commit("z060", 2, "feat: add loading skeleton", f1("src/components/Skeleton.tsx", 30, 0)),
		commit("z061", 1, "fix: skeleton layout shift", f1("src/components/Skeleton.tsx", 12, 8)),
		commit("z062", 1, "fix: dashboard memory leak", f1("src/dashboard/index.tsx", 10, 5)),
		commit("z063", 1, "hotfix: critical production error", f1("src/routes/api.ts", 20, 10)),
		commit("z064", 1, "fix: API returns 500 on valid request", f1("src/routes/api.ts", 15, 12)),
		commit("z065", 1, "revert: undo API changes", f1("src/routes/api.ts", 12, 15)),
		commit("z066", 1, "fix: restore API with proper handling", f1("src/routes/api.ts", 25, 12)),
		commit("z067", 1, "workaround: skip validation for legacy clients", f1("src/middleware/validate.ts", 15, 3)),
		commit("z068", 1, "fix: legacy workaround breaks new clients", f1("src/middleware/validate.ts", 10, 8)),
		commit("z069", 1, "feat: add fallback error page", f1("src/pages/Error.tsx", 25, 0)),
		commit("z070", 1, "fix: error page infinite redirect loop", f1("src/pages/Error.tsx", 8, 5)),
	}
}
