package ecosystem

import (
	"os/exec"
	"strings"
)

// Category distinguishes agentic stack libraries from general tooling.
type Category string

const (
	CategoryInternal Category = "internal"
	CategoryTool     Category = "tool"
)

// Library is a first-class package auracrab can install via anyisland.
type Library struct {
	Name        string
	Description string
	Category    Category
	Binary      string
	Package     string
}

// InternalLibraries are agentic stack components (no github.com/nathfavour/... URLs needed).
var InternalLibraries = []Library{
	{
		Name:        "polygeist",
		Description: "Sovereign multi-agent control plane and orchestration daemon",
		Category:    CategoryInternal,
		Binary:      "polygeist",
		Package:     "polygeist",
	},
	{
		Name:        "anyisland",
		Description: "Agentic package manager and distribution layer",
		Category:    CategoryInternal,
		Binary:      "anyisland",
		Package:     "anyisland",
	},
	{
		Name:        "vibeaura",
		Description: "CLI agentic harness (mutation engine)",
		Category:    CategoryInternal,
		Binary:      "vibeaura",
		Package:     "vibeaura",
	},
	{
		Name:        "auracrab",
		Description: "Team bridge for Telegram, Slack, and Discord",
		Category:    CategoryInternal,
		Binary:      "auracrab",
		Package:     "auracrab",
	},
}

// WellKnownTools mirrors anyisland official packages (go, node, git, etc.).
var WellKnownTools = []Library{
	{Name: "go", Description: "Go programming language", Category: CategoryTool, Binary: "go", Package: "go"},
	{Name: "node", Description: "Node.js runtime", Category: CategoryTool, Binary: "node", Package: "node"},
	{Name: "git", Description: "Git version control", Category: CategoryTool, Binary: "git", Package: "git"},
	{Name: "docker", Description: "Docker container runtime", Category: CategoryTool, Binary: "docker", Package: "docker"},
	{Name: "rust", Description: "Rust toolchain", Category: CategoryTool, Binary: "rustc", Package: "rust"},
	{Name: "python", Description: "Python runtime", Category: CategoryTool, Binary: "python3", Package: "python"},
	{Name: "redis", Description: "Redis server", Category: CategoryTool, Binary: "redis-server", Package: "redis"},
	{Name: "postgres", Description: "PostgreSQL database", Category: CategoryTool, Binary: "psql", Package: "postgres"},
	{Name: "flutter", Description: "Flutter SDK", Category: CategoryTool, Binary: "flutter", Package: "flutter"},
	{Name: "zig", Description: "Zig programming language", Category: CategoryTool, Binary: "zig", Package: "zig"},
	{Name: "ripgrep", Description: "Fast line-oriented search tool", Category: CategoryTool, Binary: "rg", Package: "ripgrep"},
	{Name: "staticcheck", Description: "Go static analysis", Category: CategoryTool, Binary: "staticcheck", Package: "staticcheck"},
	{Name: "gofumpt", Description: "Stricter gofmt", Category: CategoryTool, Binary: "gofumpt", Package: "gofumpt"},
}

// All returns internal libraries followed by well-known tools.
func All() []Library {
	out := make([]Library, 0, len(InternalLibraries)+len(WellKnownTools))
	out = append(out, InternalLibraries...)
	out = append(out, WellKnownTools...)
	return out
}

// Lookup resolves a library name (case-insensitive).
func Lookup(name string) (Library, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, lib := range All() {
		if strings.ToLower(lib.Name) == key || strings.ToLower(lib.Package) == key {
			return lib, true
		}
	}
	return Library{}, false
}

// IsInstalled reports whether the library binary is on PATH.
func IsInstalled(lib Library) bool {
	bin := lib.Binary
	if bin == "" {
		bin = lib.Name
	}
	_, err := exec.LookPath(bin)
	return err == nil
}
