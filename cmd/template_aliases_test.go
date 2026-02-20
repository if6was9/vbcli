package cmd

import "testing"

func TestSubstituteTemplateCharacterAliases(t *testing.T) {
	t.Parallel()

	got := substituteTemplateCharacterAliases("hello {green} {question mark} {66} {{props.color}} {unknown}")
	want := "hello {66} {60} {66} {{props.color}} {unknown}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSubstituteTemplateCharacterAliasesSpacing(t *testing.T) {
	t.Parallel()

	got := substituteTemplateCharacterAliases("{  purple  } {-not-known-}")
	want := "{68} {-not-known-}"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
