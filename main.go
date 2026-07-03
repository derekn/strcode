package main

import (
	"bytes"
	"encoding/ascii85"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	version     string
	decode      bool
	showVersion bool
	showList    bool
)

type codec struct {
	Encode func(string) (string, error)
	Decode func(string) (string, error)
}

var codecs = map[string]codec{
	"ascii85": {
		Encode: func(s string) (string, error) {
			var buf bytes.Buffer
			enc := ascii85.NewEncoder(&buf)
			_, err := enc.Write([]byte(s))
			if err != nil {
				return "", err
			}
			enc.Close()
			return buf.String(), nil
		},
		Decode: func(s string) (string, error) {
			dec := ascii85.NewDecoder(strings.NewReader(s))
			b, err := io.ReadAll(dec)
			return string(b), err
		},
	},
	"base32": {
		Encode: func(s string) (string, error) { return base32.StdEncoding.EncodeToString([]byte(s)), nil },
		Decode: func(s string) (string, error) {
			b, err := base32.StdEncoding.DecodeString(s)
			return string(b), err
		},
	},
	"base64": {
		Encode: func(s string) (string, error) { return base64.StdEncoding.EncodeToString([]byte(s)), nil },
		Decode: func(s string) (string, error) {
			b, err := base64.StdEncoding.DecodeString(s)
			return string(b), err
		},
	},
	"base64url": {
		Encode: func(s string) (string, error) { return base64.URLEncoding.EncodeToString([]byte(s)), nil },
		Decode: func(s string) (string, error) {
			b, err := base64.URLEncoding.DecodeString(s)
			return string(b), err
		},
	},
	"binary": {
		Encode: func(s string) (string, error) {
			parts := make([]string, len(s))
			for i, b := range []byte(s) {
				parts[i] = fmt.Sprintf("%08b", b)
			}
			return strings.Join(parts, " "), nil
		},
		Decode: func(s string) (string, error) {
			parts := strings.Fields(s)
			out := make([]byte, len(parts))
			for i, p := range parts {
				v, err := strconv.ParseUint(p, 2, 8)
				if err != nil {
					return "", fmt.Errorf("invalid binary byte %q: %w", p, err)
				}
				out[i] = byte(v)
			}
			return string(out), nil
		},
	},
	"hex": {
		Encode: func(s string) (string, error) { return hex.EncodeToString([]byte(s)), nil },
		Decode: func(s string) (string, error) {
			b, err := hex.DecodeString(s)
			return string(b), err
		},
	},
	"html": {
		Encode: func(s string) (string, error) { return html.EscapeString(s), nil },
		Decode: func(s string) (string, error) { return html.UnescapeString(s), nil },
	},
	"rot13": {
		Encode: func(s string) (string, error) { return rot13(s), nil },
		Decode: func(s string) (string, error) { return rot13(s), nil },
	},
	"rot47": {
		Encode: func(s string) (string, error) { return rot47(s), nil },
		Decode: func(s string) (string, error) { return rot47(s), nil },
	},
	"url": {
		Encode: func(s string) (string, error) { return url.QueryEscape(s), nil },
		Decode: func(s string) (string, error) { return url.QueryUnescape(s) },
	},
}

func rot13(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		default:
			return r
		}
	}, s)
}

func rot47(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '!' && r <= '~' {
			return '!' + (r-'!'+47)%94
		}
		return r
	}, s)
}

func init() {
	if version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			version = strings.TrimPrefix(info.Main.Version, "v")
		} else {
			version = time.Now().Format("2006.1.2") + "-dev"
		}
	}
	flag.BoolVar(&decode, "d", false, "decode instead of encode")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.BoolVar(&showList, "list", false, "show available formats")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: strcode [-d] <format> [input | -]\n\n")
		fmt.Fprintf(os.Stderr, "formats: %s\n\n", strings.Join(codecNames(), ", "))
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}
}

func codecNames() []string {
	names := make([]string, 0, len(codecs))
	for name := range codecs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func errExit(code int, msg string, args ...any) {
	fmt.Fprintf(os.Stderr, filepath.Base(os.Args[0])+": "+msg+"\n", args...)
	os.Exit(code)
}

func main() {
	flag.Parse()

	if showVersion {
		fmt.Printf("strcode %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		fmt.Println("Copyright (C) Derek Nicol")
		os.Exit(0)
	}
	if showList {
		for _, name := range codecNames() {
			fmt.Println(name)
		}
		os.Exit(0)
	}

	args := flag.Args()

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	format := args[0]
	c, ok := codecs[format]
	if !ok {
		errExit(1, "unknown format: %s\navailable: %s", format, strings.Join(codecNames(), ", "))
	}

	var input string
	if len(args) > 1 && (len(args) != 2 || args[1] != "-") {
		input = strings.Join(args[1:], " ")
	} else {
		fi, err := os.Stdin.Stat()
		if err != nil {
			errExit(1, "reading stdin: %v", err)
		}
		if fi.Mode()&os.ModeCharDevice == 0 {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				errExit(1, "reading stdin: %v", err)
			}
			input = strings.TrimSuffix(string(b), "\n")
		} else {
			errExit(1, "no input provided (pass as argument or pipe to stdin)")
		}
	}

	var fn func(string) (string, error)
	if decode {
		fn = c.Decode
	} else {
		fn = c.Encode
	}

	result, err := fn(input)
	if err != nil {
		errExit(1, "%s: %v", format, err)
	}
	fmt.Println(result)
}
