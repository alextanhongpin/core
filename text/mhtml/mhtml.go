package mhtml

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

// mhtmlPart represents one MIME segment (resource) extracted from the MHTML archive.
type mhtmlPart struct {
	ContentType string // The MIME type of the content (e.g., "text/html").
	Location    string // Content-Location header value (URL for resources).
	Data        []byte // Decoded body bytes of the resource data.
}

// Metadata holds the structural information extracted from the MHTML message headers.
type Metadata struct {
	URL   *url.URL  // The main content URL.
	Title string    // The subject/title of the email.
	Date  time.Time // The date the message was sent.
}

// Read handles the core reading and splitting of the multipart MHTML body.
func Read(r io.Reader, body bool) ([]byte, *Metadata, error) {
	// Read the full MIME message part.
	msg, err := mail.ReadMessage(r)
	if err != nil {
		return nil, nil, fmt.Errorf("reading mail message: %w", err)
	}

	// Extract Content-Location header, which points to the main content URL.
	u, err := url.Parse(msg.Header.Get("Snapshot-Content-Location"))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing snapshot content location header: %w", err)
	}

	date, err := msg.Header.Date()
	if err != nil {
		return nil, nil, fmt.Errorf("parsing date header: %w", err)
	}
	md := &Metadata{
		URL:   u,
		Date:  date,
		Title: msg.Header.Get("Subject"),
	}
	if !body {
		return nil, md, nil
	}

	// Determine the boundary for multipart parsing.
	contentTypeHeader := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing Content-Type '%s': %w", contentTypeHeader, err)
	}
	if mediaType != "multipart/related" {
		return nil, nil, fmt.Errorf("MHTML must be 'multipart/related', got '%s'", mediaType)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, nil, fmt.Errorf("MHTML file is missing the 'boundary' parameter")
	}

	// Initialize the multipart reader.
	mr := multipart.NewReader(msg.Body, boundary)
	var htmls []*mhtmlPart     // Stores HTML parts (the main body and iframes).
	var resources []*mhtmlPart // Stores resource parts (images, CSS, scripts).

	// Iterate over all parts in the MHTML.
	for {
		mp, err := mr.NextPart()
		if err == io.EOF {
			break // End of file reached
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading multipart part: %w", err)
		}

		ct := mp.Header.Get("Content-Type")
		cte := mp.Header.Get("Content-Transfer-Encoding")
		contentType, _, err := mime.ParseMediaType(ct)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing multipart content type [%s]: %w", ct, err)
		}
		raw, err := io.ReadAll(mp)
		if err != nil {
			return nil, nil, fmt.Errorf("reading multipart body: %w", err)
		}
		data, err := decodeTransferEncoding(raw, cte)
		if err != nil {
			return nil, nil, fmt.Errorf("decoding multipart transfer encoding [%s]: %w", cte, err)
		}

		// Determine the location/ID for this part based on MHTML structure rules.
		var location string
		if mp.Header.Get("Content-ID") != "" {
			// If Content-ID exists, it's an embedded resource (e.g., an iframe).
			location = "cid:" + strings.Trim(mp.Header.Get("Content-ID"), "<>")
		} else {
			// Otherwise, it's the main content or another object that needs to be referenced via location.
			location = mp.Header.Get("Content-Location")
		}

		p := &mhtmlPart{
			ContentType: contentType,
			Location:    location,
			Data:        data,
		}

		// Separate HTML parts from resource parts for later substitution.
		if p.ContentType == "text/html" {
			htmls = append(htmls, p)
		} else {
			resources = append(resources, p)
		}
	}

	if len(htmls)+len(resources) == 0 {
		return nil, md, fmt.Errorf("no MIME parts were successfully parsed")
	}

	// The final HTML is constructed by substituting resources into the main content.
	finalHTML := rewriteHTML(htmls, resources)

	return finalHTML, md, nil
}

// decodeTransferEncoding decodes the raw byte data based on the Transfer-Encoding header.
func decodeTransferEncoding(data []byte, enc string) ([]byte, error) {
	switch strings.ToLower(enc) {
	case "base64":
		// Decode base64 encoded data.
		return base64.StdEncoding.DecodeString(string(data))

	case "quoted-printable":
		// Decode quoted-printable encoded data.
		return io.ReadAll(bytes.NewReader(data))

	default:
		// Handle raw binary data or absence of encoding headers (7bit, 8bit, binary).
		return data, nil
	}
}

// buildResMap creates a lookup map from all resource locations/CIDs to their Data URIs.
// This prepares the embedded data for insertion into the HTML.
func buildResMap(parts []*mhtmlPart) map[string][]byte {
	rmap := make(map[string][]byte)
	for _, p := range parts { // Iterate over all parts, including the main HTML part (index 0).
		// Construct the Data URI format: data:[mime_type];base64,[data]
		contentData := base64.StdEncoding.EncodeToString(p.Data)
		rmap[p.Location] = fmt.Appendf(nil, "data:%s;base64,%s", p.ContentType, contentData)
	}
	return rmap
}

// rewriteHTML iterates through the HTML and substitutes all resource references (src, href, url())
// with their corresponding Data URIs found in the resource map.
func rewriteHTML(htmls, resources []*mhtmlPart) []byte {
	// Build the map of all resources to be used for substitution.
	for original, replacement := range buildResMap(resources) {
		for i, h := range htmls {
			htmls[i].Data = bytes.ReplaceAll(h.Data, []byte(original), replacement)
		}
	}

	// Build the map of all iframes to be used for substitution.
	main := htmls[0].Data
	iframes := htmls[1:]
	for original, replacement := range buildResMap(iframes) {
		main = bytes.ReplaceAll(main, []byte(original), replacement)
	}

	return main
}
