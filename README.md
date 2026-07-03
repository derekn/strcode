# strcode

Encode and decode strings from the command line.

## Usage

```
strcode [-d] <format> [input | -]
```

- Pass input as an argument, pipe it via stdin, or use `-` to explicitly read stdin
- Use `-d` to decode instead of encode

| Flag       | Description              |
| ---------- | ------------------------ |
| `-d`       | Decode instead of encode |
| `-list`    | List available formats   |
| `-version` | Show version             |

## Formats

ascii85, base32, base64, base64url, binary, hex, html, rot13, rot47, url

## Examples

```sh
strcode base64 "hello world"
# aGVsbG8gd29ybGQ=

strcode -d rot13 "Uryyb Jbeyq"
# Hello World
```

## Install

```sh
go install github.com/derekn/strcode@latest
```

## Building

```sh
make build
# binary written to dist/
```

Build for all platforms:

```sh
make build-all
```
