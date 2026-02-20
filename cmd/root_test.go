package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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

func TestDecodeEscapes(t *testing.T) {
	t.Parallel()

	got := decodeEscapes(`Hello\nworld`)
	want := "Hello\nworld"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDecodeEscapesInvalidSequenceFallsBack(t *testing.T) {
	t.Parallel()

	input := `Hello\qworld`
	got := decodeEscapes(input)
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
	if got := sendCmd.Flags().Lookup("format"); got == nil {
		t.Fatalf("expected --format flag for send")
	}
}

func TestGetLayoutExtraction(t *testing.T) {
	t.Parallel()

	layout, err := extractLayout([]byte(`{"currentMessage":{"layout":"1234"}}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if layout != "1234" {
		t.Fatalf("got %q, want %q", layout, "1234")
	}
}

func TestGetLayoutExtractionMissing(t *testing.T) {
	t.Parallel()

	if _, err := extractLayout([]byte(`{"currentMessage":{"id":"x"}}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTransitionType(t *testing.T) {
	t.Parallel()

	valid := []string{"classic", "wave", "drift", "curtain"}
	for _, v := range valid {
		got, err := resolveTransitionType(v)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", v, err)
		}
		if got != v {
			t.Fatalf("got %q, want %q", got, v)
		}
	}
	if _, err := resolveTransitionType("other"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTransitionSpeed(t *testing.T) {
	t.Parallel()

	got, err := resolveTransitionSpeed("fast")
	if err != nil || got != "fast" {
		t.Fatalf("got %q, err %v", got, err)
	}
	got, err = resolveTransitionSpeed("gentle")
	if err != nil || got != "gentle" {
		t.Fatalf("got %q, err %v", got, err)
	}
	if _, err := resolveTransitionSpeed("slow"); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveCommandInputFromStdinWhenArgMissing(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "send"}
	got, err := resolveCommandInput(cmd, strings.NewReader("hello from stdin\n"), nil, "message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello from stdin" {
		t.Fatalf("got %q, want %q", got, "hello from stdin")
	}
}

func TestMaxArgsWithHelp(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "send"}
	if err := maxArgsWithHelp(1)(cmd, []string{"a"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := maxArgsWithHelp(1)(cmd, []string{"a", "b"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveCommandInputMissingArgOnTerminal(t *testing.T) {
	devNull, err := os.Open("/dev/null")
	if err != nil {
		t.Skipf("unable to open /dev/null: %v", err)
	}
	defer devNull.Close()

	cmd := &cobra.Command{Use: "send"}
	if _, err := resolveCommandInput(cmd, devNull, nil, "message"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPrettyPrintJSON(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"a":1,"b":{"c":2}}`)
	got, err := prettyPrintJSON(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(got), "\n  \"a\": 1,") {
		t.Fatalf("expected indented output, got: %s", string(got))
	}
}

func TestLooksLikeRawCharactersJSON(t *testing.T) {
	t.Parallel()

	if !looksLikeRawCharactersJSON(" [[1,2],[3,4]] ") {
		t.Fatal("expected true for bracketed JSON-like input")
	}
	if looksLikeRawCharactersJSON("hello [1,2]") {
		t.Fatal("expected false when input is not fully bracketed")
	}
	if looksLikeRawCharactersJSON("{\"characters\":[[1]]}") {
		t.Fatal("expected false for object JSON")
	}
}

func TestSubcommandsExist(t *testing.T) {
	t.Parallel()

	root := NewRootCmd(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	want := map[string]bool{"send-raw": false, "send": false, "format": false, "clear": false, "get": false, "set-transition": false, "get-transition": false}
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
