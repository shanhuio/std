package dock

import (
	"reflect"
	"testing"
)

func TestShellSplit(t *testing.T) {
	ok := func(line string, want ...string) {
		t.Helper()
		got, err := shellSplit(line)
		if err != nil {
			t.Errorf("shellSplit(%q): unexpected error: %v", line, err)
			return
		}
		if len(want) == 0 && len(got) == 0 {
			return
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("shellSplit(%q): got %q, want %q", line, got, want)
		}
	}

	ok("")
	ok("a", "a")
	ok(`"a"`, "a")
	ok(`/something`, "/something")
	ok(`ls /x_file`, "ls", "/x_file")
	ok("       ls \t\t a", "ls", "a")
	ok(`"a-b" something`, "a-b", "something")
	ok(`a-b`, "a-b")
	ok(`?`, "?")
	ok(`!x`, "!x")
	ok(`mkdir -p /root/.ssh`, "mkdir", "-p", "/root/.ssh")

	ok(`"hello world"`, "hello world")
	ok(`"a b" "c d"`, "a b", "c d")
	ok(`"say \"hi\""`, `say "hi"`)
	ok(`echo "\"quoted\""`, "echo", `"quoted"`)

	fail := func(line string) {
		t.Helper()
		if _, err := shellSplit(line); err == nil {
			t.Errorf("shellSplit(%q): expected error, got nil", line)
		}
	}

	fail(`"`)
	fail(`"asdf" asdf "xx`)
}
