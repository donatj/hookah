package exec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPathExecs_PathTraversalValidation(t *testing.T) {
	data := strings.NewReader(`{"foo": "bar"}`)
	h := HookExec{
		RootDir: "../../testdata/exec-only-test-server",
		Data:    data,
	}

	tests := []struct {
		name   string
		owner  string
		repo   string
		event  string
		action string
	}{
		{"owner with slash", "user/malicious", "repo", "event", ""},
		{"repo with slash", "user", "repo/malicious", "event", ""},
		{"event with slash", "user", "repo", "event/malicious", ""},
		{"action with slash", "user", "repo", "event", "action/malicious"},
		{"owner with dot-dot", "user/..", "repo", "event", ""},
		{"repo with dot-dot", "user", "repo/..", "event", ""},
		{"event with dot-dot", "user", "repo", "event/..", ""},
		{"action with dot-dot", "user", "repo", "event", "action/.."},
		{"owner with just dot-dot", "..", "repo", "event", ""},
		{"repo with just dot-dot", "user", "..", "event", ""},
		{"event with just dot-dot", "user", "repo", "..", ""},
		{"action with just dot-dot", "user", "repo", "event", ".."},
		{"owner with dot", ".", "repo", "event", ""},
		{"empty owner", "", "repo", "event", ""},
		{"empty repo", "user", "", "event", ""},
		{"empty event", "user", "repo", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := h.GetPathExecs(tt.owner, tt.repo, tt.event, tt.action)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrPathTraversal)
		})
	}
}

func TestGetPathExecs_ValidComponents(t *testing.T) {
	data := strings.NewReader(`{"foo": "bar"}`)
	h := HookExec{
		RootDir: "../../testdata/exec-only-test-server",
		Data:    data,
	}

	tests := []struct {
		name   string
		owner  string
		repo   string
		event  string
		action string
	}{
		{"valid without action", "user", "repo", "event", ""},
		{"valid with action", "user", "repo", "event", "action"},
		{"valid with underscores", "user_name", "repo_name", "event_name", "action_name"},
		{"valid with hyphens", "user-name", "repo-name", "event-name", "action-name"},
		{"valid with numbers", "user123", "repo456", "event789", "action000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := h.GetPathExecs(tt.owner, tt.repo, tt.event, tt.action)
			assert.NoError(t, err)
		})
	}
}
