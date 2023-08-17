package hookah

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOnlyExecutableBinsFound(t *testing.T) {

	expectedScripts := []string{
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/exec-symlink.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/exec.sh",
		"testdata/exec-only-test-server/@@/exec-symlink-symlink.sh",
		"testdata/exec-only-test-server/@@/exec.sh",
		"testdata/exec-only-test-server/@@/repo/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"testdata/exec-only-test-server/user/@@/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/exec.sh",
		"testdata/exec-only-test-server/@@/@@/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/exec.sh",
	}

	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/exec-only-test-server",
		Data:    data,
	}

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "event", "")
	if err != nil {
		t.Error(err)
	}

	log.Printf("%#v", scripts)

	if !reflect.DeepEqual(scripts, expectedScripts) {
		t.Errorf("expected\n%#v\n\ngot\n%#v", expectedScripts, scripts)
	}

	if !reflect.DeepEqual(errhandlers, expectedErrhandlers) {
		t.Errorf("expected\n%#v\n\ngot\n%#v", expectedErrhandlers, errhandlers)
	}
}

func TestActionDirectoriesWorkAsExpected(t *testing.T) {

	expectedScripts := []string{
		"testdata/exec-only-test-server/exec.sh",
		"testdata/exec-only-test-server/exec-symlink.sh",
		"testdata/exec-only-test-server/user/exec.sh",
		"testdata/exec-only-test-server/user/repo/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/exec.sh",
		"testdata/exec-only-test-server/user/repo/event/action/exec.sh",
		"testdata/exec-only-test-server/@@/exec-symlink-symlink.sh",
		"testdata/exec-only-test-server/@@/exec.sh",
		"testdata/exec-only-test-server/@@/repo/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/action/exec.sh",
		"testdata/exec-only-test-server/user/@@/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/exec.sh",
		"testdata/exec-only-test-server/user/@@/event/action/exec.sh",
		"testdata/exec-only-test-server/@@/@@/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/action/exec.sh",
	}
	expectedErrhandlers := []string{
		"testdata/exec-only-test-server/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/@@error.exec.sh",
		"testdata/exec-only-test-server/user/repo/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/@@/repo/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/user/@@/event/action/@@error.exec.sh",
		"testdata/exec-only-test-server/@@/@@/event/action/@@error.exec.sh",
	}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/exec-only-test-server",
		Data:    data,
	}

	scripts, errhandlers, err := h.GetPathExecs("user", "repo", "event", "action")
	if err != nil {
		t.Error(err)
	}

	log.Printf("%#v", scripts)

	if !reflect.DeepEqual(scripts, expectedScripts) {
		t.Errorf("expected\n%#v\n\ngot\n%#v", expectedScripts, scripts)
	}

	if !reflect.DeepEqual(errhandlers, expectedErrhandlers) {
		t.Errorf("expected\n%#v\n\ngot\n%#v", expectedErrhandlers, errhandlers)
	}
}

func TestEnvPopulatedCorrectly(t *testing.T) {

	out := &bytes.Buffer{}

	data := strings.NewReader(`{"foo": "bar"}`)

	h := HookExec{
		RootDir: "./testdata/env-test-server",
		Data:    data,

		Stdout: out,
	}

	err := h.Exec("user", "repo", "event", "action", 1*time.Minute, "FOO=BAR", "BAZ=QUX")
	if err != nil {
		t.Error(err)
	}

	env := out.String()
	lines := strings.Split(strings.TrimSpace(env), "\n")
	envMap := make(map[string]string, len(lines))

	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		envMap[parts[0]] = parts[1]
	}

	expectedEnv := map[string]string{
		"FOO": "BAR",
		"BAZ": "QUX",
	}

	for k, expectedV := range expectedEnv {
		if actualV, ok := envMap[k]; !ok || actualV != expectedV {
			t.Error("expected", k, "to be", expectedV, "got", actualV)
		}
	}

}
