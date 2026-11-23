package mhtml

import (
	"cmp"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"
)

// part is one MIME body part of the archive.
type part struct {
	contentType string   // full media type, parameters included
	html        bool     // media type is text/html
	refs        []string // "cid:..." and Content-Location values that reference it
	data        []byte   // decoded body
}

// Metadata holds the structural information extracted from the MHTML message headers.
type Metadata struct {
	URL   *url.URL  // Snapshot-Content-Location; empty if absent.
	Title string    // Subject, with RFC 2047 encoded-words decoded.
	Date  time.Time // Date; zero if absent.
}

// Read parses an MHTML (RFC 2557) archive. It always returns the metadata;
// when body is true it also returns the root HTML document with its
// resources and frames inlined as data: URIs.
//
// The root is the part named by the multipart/related "start" parameter
// (RFC 2387), or else the first part. Parts are referenced by their
// Content-Location and by "cid:" plus their Content-ID. References are
// replaced literally, longest first, in one pass; this is not an HTML parser,
// so relative URLs are not resolved. Frames are inlined one level deep.
func Read(r io.Reader, body bool) ([]byte, *Metadata, error) {
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, nil, fmt.Errorf("reading mail message: %w", err)
	}

	u, err := url.Parse(msg.Header.Get("Snapshot-Content-Location"))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing snapshot content location header: %w", err)
	}

	var date time.Time
	if msg.Header.Get("Date") != "" {
		date, err = msg.Header.Date()
		if err != nil {
			return nil, nil, fmt.Errorf("parsing date header: %w", err)
		}
	}

	title := msg.Header.Get("Subject")
	if decoded, err := new(mime.WordDecoder).DecodeHeader(title); err == nil {
		title = decoded
	}

	md := &Metadata{URL: u, Date: date, Title: title}
	if !body {
		return nil, md, nil
	}

	contentTypeHeader := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing Content-Type %q: %w", contentTypeHeader, err)
	}
	if mediaType != "multipart/related" {
		return nil, nil, fmt.Errorf("MHTML must be multipart/related, got %q", mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, nil, errors.New("MHTML file is missing the boundary parameter")
	}
	start := strings.Trim(params["start"], "<>")

	parts, root, err := readParts(multipart.NewReader(msg.Body, boundary), start)
	if err != nil {
		return nil, nil, err
	}
	if root < 0 {
		return nil, nil, fmt.Errorf("start part %q not found", start)
	}
	if !parts[root].html {
		return nil, nil, fmt.Errorf("root part is %s, not text/html", parts[root].contentType)
	}

	return inline(parts, root), md, nil
}

// readParts decodes every part and returns the index of the root part.
func readParts(mr *multipart.Reader, start string) ([]*part, int, error) {
	var parts []*part
	root := -1
	for {
		// NextPart decodes quoted-printable bodies itself.
		mp, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, -1, fmt.Errorf("reading multipart part: %w", err)
		}

		ct := mp.Header.Get("Content-Type")
		if ct == "" {
			ct = "text/plain" // RFC 2045 default
		}
		mediaType, params, err := mime.ParseMediaType(ct)
		if err != nil {
			return nil, -1, fmt.Errorf("parsing multipart content type %q: %w", ct, err)
		}

		var src io.Reader = mp
		cte := strings.ToLower(strings.TrimSpace(mp.Header.Get("Content-Transfer-Encoding")))
		if cte == "base64" {
			src = base64.NewDecoder(base64.StdEncoding, mp)
		}
		data, err := io.ReadAll(src)
		if err != nil {
			return nil, -1, fmt.Errorf("reading multipart body (%s): %w", cmp.Or(cte, "7bit"), err)
		}

		p := &part{
			// data: URIs separate parameters with ";" and no spaces.
			contentType: strings.ReplaceAll(mime.FormatMediaType(mediaType, params), "; ", ";"),
			html:        mediaType == "text/html",
			data:        data,
		}
		cid := strings.Trim(strings.TrimSpace(mp.Header.Get("Content-ID")), "<>")
		if cid != "" {
			p.refs = append(p.refs, "cid:"+cid)
		}
		if loc := strings.TrimSpace(mp.Header.Get("Content-Location")); loc != "" {
			p.refs = append(p.refs, loc)
		}
		if (start == "" && len(parts) == 0) || (start != "" && cid == start) {
			root = len(parts)
		}
		parts = append(parts, p)
	}
	if len(parts) == 0 {
		return nil, -1, errors.New("no MIME parts found")
	}
	return parts, root, nil
}

// inline returns the root document with references to the other parts
// replaced by data: URIs. Resources are first inlined into frames, then
// resources and frames into the root.
func inline(parts []*part, root int) []byte {
	var resources, frames []*part
	for i, p := range parts {
		switch {
		case i == root:
		case p.html:
			frames = append(frames, p)
		default:
			resources = append(resources, p)
		}
	}

	resolve := replacer(resources)
	for _, f := range frames {
		f.data = []byte(resolve.Replace(string(f.data)))
	}
	return []byte(replacer(append(resources, frames...)).Replace(string(parts[root].data)))
}

// replacer maps each reference to its part's data: URI. Longer references
// come first so that a URL is never replaced by the entry for its prefix.
func replacer(parts []*part) *strings.Replacer {
	type pair struct{ ref, uri string }
	var pairs []pair
	seen := make(map[string]bool)
	for _, p := range parts {
		uri := "data:" + p.contentType + ";base64," + base64.StdEncoding.EncodeToString(p.data)
		for _, ref := range p.refs {
			if !seen[ref] { // first part with a reference wins
				seen[ref] = true
				pairs = append(pairs, pair{ref, uri})
			}
		}
	}
	slices.SortStableFunc(pairs, func(a, b pair) int { return cmp.Compare(len(b.ref), len(a.ref)) })

	oldnew := make([]string, 0, 2*len(pairs))
	for _, p := range pairs {
		oldnew = append(oldnew, p.ref, p.uri)
	}
	return strings.NewReplacer(oldnew...)
}
