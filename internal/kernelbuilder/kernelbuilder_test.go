package kernelbuilder

import "testing"

func TestLastLines(t *testing.T) {
	if got := lastLines("", 5); got != "" {
		t.Errorf("empty input should stay empty, got %q", got)
	}
	if got := lastLines("a\nb\nc", 5); got != "a; b; c" {
		t.Errorf("short input should pass through, got %q", got)
	}
	// Fehlermeldung darf nicht den ganzen Output verschlucken
	if got := lastLines("1\n2\n3\n4\n5\n6", 3); got != "4; 5; 6" {
		t.Errorf("expected last 3 lines, got %q", got)
	}
}

// Auf diesem Host muss mindestens eine Abfrage laufen können
func TestMissingDependencies(t *testing.T) {
	missing, err := missingDependencies()
	if err != nil {
		t.Skipf("no supported package manager: %v", err)
	}
	if len(missing) > 0 {
		t.Errorf("build deps reported missing on a host that has them: %v", missing)
	}
}

// dependencyQuery muss eine Abfrage liefern, keine Installation.
func TestDependencyQuery(t *testing.T) {
	deps, query, err := dependencyQuery()
	if err != nil {
		t.Skipf("neither rpm nor dpkg-query available: %v", err)
	}
	if len(deps) == 0 || len(query) < 2 {
		t.Fatalf("got deps=%v query=%v", deps, query)
	}
	for _, arg := range query {
		if arg == "install" || arg == "-y" || arg == "update" {
			t.Errorf("query must not install anything, got %q", arg)
		}
	}
}
