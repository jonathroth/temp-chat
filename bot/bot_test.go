package bot

import "testing"

func TestBad(t *testing.T) {
	t.Log("A Log message")
	t.Fatal("Failed test")
}
