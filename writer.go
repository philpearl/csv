package csv

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Writer is used to write to a CSV file.
type Writer struct {
	w io.Writer
	b []byte
}

// NewWriter creates a new CSV writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w: w,
	}
}

// String writes a string cell value to the CSV. It escapes the string value if necessary
func (w *Writer) String(s string) {
	w.comma()
	if !w.fieldNeedsQuotes(s) {
		w.b = append(w.b, s...)
		return
	}
	w.b = append(w.b, '"')
	// If we range through a string by value we'll be given runes. But we don't need runes as we only need to
	// look for ", and no byte of a utf8 char will match unless it is a "
	for i := range s {
		c := s[i]
		switch c {
		case '"':
			w.b = append(w.b, '"', '"')
		default:
			// Even other special characters are just copied
			w.b = append(w.b, c)
		}
	}
	w.b = append(w.b, '"')
}

// Bytes writes a []byte as a cell value to the CSV. The []byte is assumed to be a string. It is used where
// the caller has a []byte for this cell and not converting to a string is more efficient
func (w *Writer) Bytes(s []byte) {
	w.comma()
	if !w.byteFieldNeedsQuotes(s) {
		w.b = append(w.b, s...)
		return
	}
	w.b = append(w.b, '"')
	// If we range through a string by value we'll be given runes. But we don't need runes as we only need to
	// look for ", and no byte of a utf8 char will match unless it is a "
	for i := range s {
		c := s[i]
		switch c {
		case '"':
			w.b = append(w.b, '"', '"')
		default:
			// Even other special characters are just copied
			w.b = append(w.b, c)
		}
	}
	w.b = append(w.b, '"')
}

// Float64 writes a float64 cell value to the CSV
func (w *Writer) Float64(f float64) {
	w.comma()
	w.b = strconv.AppendFloat(w.b, f, 'g', -1, 64)
}

// Int64 writes an int64 cell value to the CSV
func (w *Writer) Int64(i int64) {
	w.comma()
	w.b = strconv.AppendInt(w.b, i, 10)
}

// LineComplete finishes the CSV file line and writes it to the output
func (w *Writer) LineComplete() error {
	w.b = append(w.b, '\n')
	_, err := w.w.Write(w.b)
	w.b = w.b[:0]
	return err
}

func (w *Writer) comma() {
	if len(w.b) != 0 {
		w.b = append(w.b, ',')
	}
}

// fieldNeedsQuotes reports whether our field must be enclosed in quotes.
// Fields with a Comma, fields with a quote or newline, and
// fields which start with a space must be enclosed in quotes.
// We used to quote empty strings, but we do not anymore (as of Go 1.4).
// The two representations should be equivalent, but Postgres distinguishes
// quoted vs non-quoted empty string during database imports, and it has
// an option to force the quoted behavior for non-quoted CSV but it has
// no option to force the non-quoted behavior for quoted CSV, making
// CSV with quoted empty strings strictly less useful.
// Not quoting the empty string also makes this package match the behavior
// of Microsoft Excel and Google Drive.
// For Postgres, quote the data terminating string `\.`.
//
// Lifted from the Go source
func (*Writer) fieldNeedsQuotes(field string) bool {
	if field == "" {
		return false
	}
	if field == `\.` || strings.ContainsRune(field, ',') || strings.ContainsAny(field, "\"\r\n") {
		return true
	}

	r1, _ := utf8.DecodeRuneInString(field)
	return unicode.IsSpace(r1)
}

func (*Writer) byteFieldNeedsQuotes(field []byte) bool {
	if len(field) == 0 {
		return false
	}
	if bytes.ContainsRune(field, ',') || bytes.ContainsAny(field, "\"\r\n") {
		return true
	}
	if len(field) == 2 && field[0] == '\\' && field[1] == '.' {
		return true
	}

	r1, _ := utf8.DecodeRune(field)
	return unicode.IsSpace(r1)
}
