# mhtml

Read metadata and a self-contained HTML document from MHTML (RFC 2557)
archives, such as Chrome's "Save page as… Webpage, Single File".

```sh
go get github.com/alextanhongpin/core/text/mhtml
```

```go
f, err := os.Open("snapshot.mhtml")
if err != nil {
	return err
}
defer f.Close()

html, md, err := mhtml.Read(f, true) // false: metadata only
if err != nil {
	return err
}
fmt.Println(md.URL, md.Title, md.Date, len(html))
```

## Behavior

- Metadata comes from the top-level headers: `Snapshot-Content-Location`,
  `Subject` (RFC 2047 encoded-words decoded) and `Date`. A missing `Date`
  gives the zero time; an invalid one is an error.
- The body must be `multipart/related` with a boundary. The root part is the
  one whose Content-ID matches the `start` parameter (RFC 2387), else the
  first part, and it must be `text/html`.
- Base64 and quoted-printable parts are decoded with the standard library.
- Every other part is referenced by its `Content-Location` and by
  `cid:` + its Content-ID. Those references in the root are replaced with
  `data:` URIs in a single pass, longest reference first; frames (other HTML
  parts) get their resources inlined first, one level deep.

Replacement is literal text substitution, not HTML parsing: relative URLs
are not resolved and the output is not sanitized. The whole archive is held
in memory.
