package grpcdump

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	linePrefix    = "GRPC "
	separator     = "=== "
	statusPrefix  = "=== status"
	clientPrefix  = "=== client"
	serverPrefix  = "=== server"
	headerPrefix  = "=== header"
	trailerPrefix = "=== trailer"
)

var ErrInvalidDumpFormat = errors.New("grpcdump: invalid dump format")

// https://github.com/bradleyjkemp/grpc-tools/blob/master/grpc-dump/README.md
type Dump struct {
	Addr       string      `json:"addr"`
	FullMethod string      `json:"full_method"`
	Messages   []Message   `json:"messages"`
	Status     *Status     `json:"status"`
	Metadata   metadata.MD `json:"metadata"` // The server receives metadata.
	Header     metadata.MD `json:"header"`   // The client receives header and trailer.
	Trailer    metadata.MD `json:"trailer"`
}

func (d *Dump) Service() string {
	return filepath.Dir(d.FullMethod)
}

func (d *Dump) Method() string {
	return filepath.Base(d.FullMethod)
}

func (d *Dump) AsText() ([]byte, error) {
	sb := new(strings.Builder)

	sb.WriteString(writeMethod(d.Addr, d.FullMethod))
	sb.WriteRune('\n')

	writeMetadata(sb, "", d.Metadata)
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	sb.WriteString("###")
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteRune('\n')

	// Header is written before any response.
	writeMetadata(sb, headerPrefix, d.Header)
	sb.WriteRune('\n')

	if err := writeMessages(sb, d.Messages...); err != nil {
		return nil, err
	}

	// Status is before trailer.
	if err := writeStatus(sb, d.Status); err != nil {
		return nil, err
	}
	sb.WriteRune('\n')

	// Trailer is the optional, and is the last to be sent.
	writeMetadata(sb, trailerPrefix, d.Trailer)

	return []byte(sb.String()), nil
}

func (d *Dump) FromText(b []byte) error {
	b = bytes.TrimLeft(b, " \t\r\n")

	scanner := bufio.NewScanner(bytes.NewReader(b))

	if !scanner.Scan() {
		return ErrInvalidDumpFormat
	}

	text := scanner.Text()
	if !strings.HasPrefix(text, linePrefix) {
		return fmt.Errorf("%w: invalid line %q", ErrInvalidDumpFormat, text)
	}

	text = strings.TrimPrefix(text, linePrefix)
	addr, fullMethod, ok := strings.Cut(text, "/")
	if !ok {
		return fmt.Errorf("%w: invalid line %q", ErrInvalidDumpFormat, text)
	}

	d.Addr = addr
	d.FullMethod = fullMethod

	md, err := scanMetadata(scanner)
	if err != nil {
		return err
	}
	d.Metadata = md

	// Scan server and client messages.
	for scanner.Scan() {
		text := scanner.Text()
		if !strings.HasPrefix(text, separator) {
			continue
		}

		switch {
		case
			strings.HasPrefix(text, clientPrefix),
			strings.HasPrefix(text, serverPrefix):

			b := scanBody(scanner)

			var a any
			if err := json.Unmarshal(b, &a); err != nil {
				return err
			}

			text = strings.TrimPrefix(text, separator)
			origin, name, ok := strings.Cut(text, ": ")
			if !ok {
				return ErrInvalidDumpFormat
			}

			d.Messages = append(d.Messages, Message{
				Origin:  origin,
				Message: a,
				Name:    name,
			})
		case text == statusPrefix:
			b := scanBody(scanner)

			var e Status
			if err := json.Unmarshal(b, &e); err != nil {
				return err
			}

			d.Status = &e
		case text == headerPrefix:
			header, err := scanMetadata(scanner)
			if err != nil {
				return err
			}
			d.Header = header

		case text == trailerPrefix:
			trailer, err := scanMetadata(scanner)
			if err != nil {
				return err
			}
			d.Trailer = trailer
		}
	}

	return nil
}

func writeStatus(sb *strings.Builder, status *Status) error {
	if status == nil {
		return nil
	}

	b, err := json.MarshalIndent(status, "", " ")
	if err != nil {
		return err
	}
	sb.WriteString(statusPrefix)
	sb.WriteRune('\n')
	sb.Write(b)
	sb.WriteRune('\n')
	return nil
}

func writeMethod(addr, fullMethod string) string {
	return fmt.Sprintf("%s%s", linePrefix, filepath.Join(addr, fullMethod))
}

func writeMetadata(sb *strings.Builder, prefix string, md metadata.MD) {
	if prefix != "" && len(md) > 0 {
		sb.WriteString(prefix)
		sb.WriteRune('\n')
	}

	for k, vs := range md {
		fn := func(s string) string {
			if isBinHeader(k) {
				// https://github.com/grpc/grpc-go/pull/1209/files
				return encodeBinHeader([]byte(s))
			}

			return s
		}

		for _, v := range vs {
			sb.WriteString(fmt.Sprintf("%s: %s", k, fn(v)))
			sb.WriteRune('\n')

		}
	}
}

func writeMessages(sb *strings.Builder, msgs ...Message) error {
	for _, msg := range msgs {
		b, err := json.MarshalIndent(msg.Message, "", " ")
		if err != nil {
			return err
		}

		var prefix string
		if msg.Origin == OriginServer {
			prefix = serverPrefix
		} else {
			prefix = clientPrefix
		}
		header := fmt.Sprintf("%s: %s", prefix, msg.Name)

		sb.WriteString(header)
		sb.WriteRune('\n')
		sb.Write(b)
		sb.WriteRune('\n')
		sb.WriteRune('\n')
	}

	return nil
}

func encodeBinHeader(v []byte) string {
	return base64.RawStdEncoding.EncodeToString(v)
}

func decodeBinHeader(v string) ([]byte, error) {
	if len(v)%4 == 0 {
		// Input was padded, or padding was not necessary.
		return base64.StdEncoding.DecodeString(v)
	}

	return base64.RawStdEncoding.DecodeString(v)
}

func isBinHeader(key string) bool {
	return strings.HasSuffix(key, "-bin")
}

func scanBody(scanner *bufio.Scanner) []byte {
	var s []string
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			break
		}

		s = append(s, text)
	}

	return []byte(strings.Join(s, "\n"))
}

func scanMetadata(scanner *bufio.Scanner) (metadata.MD, error) {
	m := make(map[string]string)
	for scanner.Scan() {
		text := scanner.Text()
		if len(text) == 0 {
			break
		}

		k, v, ok := strings.Cut(text, ": ")
		if !ok {
			return nil, fmt.Errorf("%w: invalid metadata %q", ErrInvalidDumpFormat, text)
		}

		if isBinHeader(k) {
			b, err := decodeBinHeader(v)
			if err != nil {
				return nil, err
			}
			v = string(b)
		}

		m[k] = v
	}

	return metadata.New(m), nil
}
