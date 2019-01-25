package hookah

import (
	"reflect"
	"strings"
	"testing"
)

func TestOnlyExecutableBinsFound(t *testing.T) {
	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/exec-only-test-server",
		Data:    data,
	}

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "pull_request_review_comment")
	if err != nil {
		t.Error(err)
	}

	expectedScripts := []string{
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
	}

	if !reflect.DeepEqual(scripts, expectedScripts) {
		t.Errorf("expected %#v; got %#v", scripts, expectedScripts)
	}

	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
	}

	if !reflect.DeepEqual(errhandlers, expectedErrhandlers) {
		t.Errorf("expected %#v; got %#v", errhandlers, expectedErrhandlers)
	}

}
