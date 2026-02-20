package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveValue(t *testing.T) {
	t.Parallel()

	got, err := resolveValue(strings.NewReader(" from stdin \n"), "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from stdin" {
		t.Fatalf("got %q, want %q", got, "from stdin")
	}
}

func TestParseCharacters(t *testing.T) {
	t.Parallel()

	chars, err := parseCharacters("[[1,2],[3,4]]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chars) != 2 || len(chars[0]) != 2 || chars[1][1] != 4 {
		t.Fatalf("unexpected parsed value: %#v", chars)
	}

	if _, err := parseCharacters("{\"characters\":[[1]]}"); err == nil {
		t.Fatalf("expected parse failure for non-array payload")
	}
}

func TestResolveModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", input: "", want: "flagship"},
		{name: "flagship", input: "flagship", want: "flagship"},
		{name: "note", input: "note", want: "note"},
		{name: "case insensitive", input: "NoTe", want: "note"},
		{name: "invalid", input: "other", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveModel(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveModelFromEnv(t *testing.T) {
	t.Setenv(envVestaboardModel, "note")
	got, err := resolveModel("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "note" {
		t.Fatalf("got %q, want %q", got, "note")
	}
}

func TestResolveModelFlagOverridesEnv(t *testing.T) {
	t.Setenv(envVestaboardModel, "note")
	got, err := resolveModel("flagship")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "flagship" {
		t.Fatalf("got %q, want %q", got, "flagship")
	}
}

func TestResolveModelInvalidEnv(t *testing.T) {
	t.Setenv(envVestaboardModel, "invalid")
	if _, err := resolveModel(""); err == nil {
		t.Fatalf("expected error for invalid %s", envVestaboardModel)
	}
}

func TestResolveAlign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", input: "", want: "center"},
		{name: "top", input: "top", want: "top"},
		{name: "bottom", input: "bottom", want: "bottom"},
		{name: "case insensitive", input: "CeNtEr", want: "center"},
		{name: "invalid", input: "left", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveAlign(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveJustify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", input: "", want: "center"},
		{name: "left", input: "left", want: "left"},
		{name: "right", input: "right", want: "right"},
		{name: "justified", input: "justified", want: "justified"},
		{name: "case insensitive", input: "CeNtEr", want: "center"},
		{name: "invalid", input: "top", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveJustify(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDecodeTemplateEscapes(t *testing.T) {
	t.Parallel()

	got := decodeTemplateEscapes(`Hello\nworld`)
	want := "Hello\nworld"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDecodeTemplateEscapesInvalidSequenceFallsBack(t *testing.T) {
	t.Parallel()

	input := `Hello\qworld`
	got := decodeTemplateEscapes(input)
	if got != input {
		t.Fatalf("got %q, want %q", got, input)
	}
}

func TestTemplateLayoutFlagShorthands(t *testing.T) {
	t.Parallel()

	root := NewRootCmd(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})

	sendCmd, _, err := root.Find([]string{"send"})
	if err != nil {
		t.Fatalf("failed to find send command: %v", err)
	}
	if got := sendCmd.Flags().ShorthandLookup("a"); got == nil || got.Name != "align" {
		t.Fatalf("expected -a shorthand for --align")
	}
	if got := sendCmd.Flags().ShorthandLookup("j"); got == nil || got.Name != "justify" {
		t.Fatalf("expected -j shorthand for --justify")
	}
}

func TestSubcommandsExist(t *testing.T) {
	t.Parallel()

	root := NewRootCmd(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	want := map[string]bool{"send-raw": false, "send": false}
	for _, sub := range root.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("missing subcommand %q", name)
		}
	}
}
