package runner

import "testing"

func TestJobSlug(t *testing.T) {
	if got := jobSlug("Consum España"); got != "consum_espana" { t.Fatalf("got %q", got) }
}
