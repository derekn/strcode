package main

import (
	"testing"
	"unicode/utf8"
)

func TestCodecs(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		input   string
		encoded string
	}{
		{"base64", "base64", "hello world", "aGVsbG8gd29ybGQ="},
		{"base64 empty", "base64", "", ""},
		{"base64url", "base64url", "hello world", "aGVsbG8gd29ybGQ="},
		{"base64url url-safe chars", "base64url", "\xfb\xff\xfe", "-__-"},
		{"base32", "base32", "hello world", "NBSWY3DPEB3W64TMMQ======"},
		{"hex", "hex", "cafe", "63616665"},
		{"hex binary", "hex", "\x00\xff", "00ff"},
		{"url", "url", "hello world!", "hello+world%21"},
		{"url special", "url", "a=1&b=2", "a%3D1%26b%3D2"},
		{"rot13 alpha", "rot13", "Hello World", "Uryyb Jbeyq"},
		{"rot13 non-alpha", "rot13", "123!@#", "123!@#"},
		{"rot47", "rot47", "Hello World!", "w6==@ (@C=5P"},
		{"rot47 passthrough", "rot47", "hello\t\n", "96==@\t\n"},
		{"html", "html", `<script>alert("xss")</script>`, "&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;"},
		{"html ampersand", "html", "A & B", "A &amp; B"},
		{"ascii85", "ascii85", "hello", "BOu!rDZ"},
		{"binary", "binary", "AB", "01000001 01000010"},
		{"binary space", "binary", " ", "00100000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := codecs[tt.format]

			got, err := c.Encode(tt.input)
			if err != nil {
				t.Fatalf("encode error: %v", err)
			}
			if got != tt.encoded {
				t.Errorf("encode: got %q, want %q", got, tt.encoded)
			}

			decoded, err := c.Decode(tt.encoded)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if decoded != tt.input {
				t.Errorf("decode: got %q, want %q", decoded, tt.input)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	inputs := []string{
		"hello world",
		"",
		"The quick brown fox jumps over the lazy dog",
		"special chars: !@#$%^&*()_+-=[]{}|;':\",./<>?",
		"\x00\x01\x02\xff",
		"unicode: éèê 世界",
	}

	for format, c := range codecs {
		for _, input := range inputs {
			if (format == "rot13" || format == "rot47" || format == "html") && !utf8.ValidString(input) {
				continue
			}
			t.Run(format+"/"+input, func(t *testing.T) {
				encoded, err := c.Encode(input)
				if err != nil {
					t.Fatalf("encode error: %v", err)
				}
				decoded, err := c.Decode(encoded)
				if err != nil {
					t.Fatalf("decode error: %v", err)
				}
				if decoded != input {
					t.Errorf("round-trip failed: got %q, want %q", decoded, input)
				}
			})
		}
	}
}

func TestCodecNames(t *testing.T) {
	names := codecNames()
	if len(names) != len(codecs) {
		t.Errorf("got %d names, want %d", len(names), len(codecs))
	}
	for i := 1; i < len(names); i++ {
		if names[i] <= names[i-1] {
			t.Errorf("names not sorted: %q after %q", names[i], names[i-1])
		}
	}
}

func TestRot13(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz", "NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm"},
		{"Hello, World!", "Uryyb, Jbeyq!"},
		{"", ""},
		{"12345", "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := rot13(tt.input)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if rot13(got) != tt.input {
				t.Error("double rot13 did not return original")
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	tests := []struct {
		name   string
		format string
		input  string
	}{
		{"base64 invalid", "base64", "not-valid-base64!!!"},
		{"base32 invalid", "base32", "not-valid-base32!!!"},
		{"hex odd length", "hex", "abc"},
		{"hex invalid chars", "hex", "xyz"},
		{"binary invalid", "binary", "00000000 notbinary"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := codecs[tt.format]
			_, err := c.Decode(tt.input)
			if err == nil {
				t.Error("expected decode error, got nil")
			}
		})
	}
}
