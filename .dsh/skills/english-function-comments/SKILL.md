---
name: english-function-comments
description: Write a Go doc comment in English for every new or modified function. Use when adding functions, methods, or exported types to this repository, or when asked to document them.
---

# English function comments

Every new or modified function, method, and exported type in this repository gets a doc
comment **in English**, written directly above the declaration. Existing Russian comments
are left untouched unless you are already editing that declaration.

## Why English

The README documents every function in Russian for the team, but the source comments are
English. That keeps the code readable to anyone who opens the file and keeps identifiers,
types, and prose in one language. Do not translate the README to match, and do not write
new Russian comments in `.go` files.

## Format

A doc comment starts with the identifier it describes and reads as a sentence:

```go
// DecompressGzipBody decompresses gzip-encoded data.
// If the payload is not valid gzip, the original body is returned unchanged.
func DecompressGzipBody(body []byte) ([]byte, error) {
```

Rules that follow Go convention and this repository's style:

- Start with the exact declared name: `// GetMangaByID returns ...`, not `// Returns ...`.
- Use a full sentence with a capital letter and a trailing period.
- Say what the function does, then — only if non-obvious — what it returns or how it fails.
- Describe behavior a caller cannot infer from the signature: fallbacks, silent errors,
  sleeping, deduplication, side effects on the database or Telegram.
- Keep it to one to three lines. A wall of prose helps nobody.
- Do not restate the signature (`// takes a string and returns an error` is noise).
- Do not number the paragraphs, and do not use a `Принимает` / `Возвращает` block — that
  shape belongs to the README, not to source comments.

For a method, name the receiver type only when the sentence would otherwise be ambiguous:

```go
// IsValid reports whether the source name is a supported parser.
func (s SourceName) IsValid() bool {
```

For a struct field or a constant group, a short trailing comment is enough:

```go
const (
    SourceReadmanga SourceName = "readmanga" // Readmanga mirror
    SourceMintmanga SourceName = "mintmanga" // Mintmanga mirror
)
```

## Behavior worth calling out

This codebase has several places where the signature hides the real behavior. When you
touch these, the comment must state the actual behavior:

- A function returning `error` that never actually returns a non-nil error — say so.
- `FindRSSLink` performs an HTTP request; `ReadmangaParser` sleeps 4 seconds between manga
  and swallows per-manga failures instead of returning them.
- `CreateChapter` returns `(nil, nil)` when the chapter already exists, so callers see
  "no error and no value" rather than a conflict.
- `TransformRSSFeed` silently caps the chapter list at 10 items.

## Adding a new function

1. Write the function.
2. Add the English doc comment above it, following the format above.
3. Run `gofmt -l .` and `go vet ./...`; both must be clean.
4. If the function is exported and part of a documented package, add it to `README.md`
   under the matching package section, in Russian, using the existing
   `Принимает` / `Возвращает` format. The README is the Russian reference; the source
   comment is English. Both exist on purpose.

## Do not

- Leave a new function without a comment, even a short unexported helper.
- Write the comment in Russian, or mix languages in one comment.
- Add a comment that repeats the function name and nothing else
  (`// handleAdd handles add.` adds no information).
- Rewrite or translate existing Russian comments that you are not otherwise changing.
