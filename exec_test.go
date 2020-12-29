package hookah

import (
	"log"
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

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "event")
	if err != nil {
		t.Error(err)
	}

	log.Printf("%#v", scripts)

	expectedScripts := []string{
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/exec.sh",
		"testdata/exec-only-test-server/@@/exec.sh",
		"testdata/exec-only-test-server/@@/repo/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"testdata/exec-only-test-server/user/@@/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/exec.sh",
		"testdata/exec-only-test-server/@@/@@/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/exec.sh",
	}

	if !reflect.DeepEqual(scripts, expectedScripts) {
		t.Errorf("expected %#v; got %#v", expectedScripts, scripts)
	}

	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
	}

	if !reflect.DeepEqual(errhandlers, expectedErrhandlers) {
		t.Errorf("expected %#v; got %#v", expectedErrhandlers, errhandlers)
	}

}
