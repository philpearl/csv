package csv

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"unsafe"
)

var (
	// ErrRowDone is returned when trying to read a cell from a row when there are no more cells to read. Call
	// Scan to access the next row.
	ErrRowDone = errors.New("row is already complete")
)

// Reader is a CSV file reader. Create with NewReader.
type Reader struct {
	r   io.Reader
	buf []byte
	pos int

	cell     []byte
	rowDone  bool
	fileDone bool
}

// NewReader creates a new CSV file reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:   r,
		buf: make([]byte, 0, 4096*8),
	}
}

// SetInput lets you use an existing Reader with a new input file.
func (r *Reader) SetInput(in io.Reader) {
	r.r = in
	r.pos = 0
	r.buf = r.buf[:0]
	r.rowDone = false
	r.fileDone = false
}

//go:generate stringer -type cellState
type cellState byte

const (
	cellStateBegin cellState = iota
	cellStateInQuote
	cellStateInQuoteQuote
	cellStateInCell
	cellStateTrailingWhiteSpace
	cellStateSlashR
)

// Scan returns false if the CSV file is done
func (r *Reader) Scan() bool {
	r.rowDone = false
	return !r.fileDone
}

// ScanLine returns false if the current row of the CSV file is done. Call it between calls to Bytes
func (r *Reader) ScanLine() bool {
	return !r.rowDone
}

// Int reads the next cell as an int
func (r *Reader) Int() (int, error) {
	if r.rowDone {
		return 0, ErrRowDone
	}
	b, err := r.Bytes()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(*(*string)(unsafe.Pointer(&b)))
}

// Float reads the next cell as a float.
func (r *Reader) Float() (float64, error) {
	if r.rowDone {
		return 0, ErrRowDone
	}
	b, err := r.Bytes()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 64)
}

// Read returns the entire next line of the CSV file as a []string
func (r *Reader) Read() ([]string, error) {
	var row []string
	for r.ScanLine() {
		t, err := r.Text()
		if err != nil {
			return row, err
		}
		row = append(row, t)
	}
	return row, nil
}

// Text returns the next cell in the CSV as a string
func (r *Reader) Text() (string, error) {
	b, err := r.Bytes()
	return string(b), err
}

// Bytes returns the next cell in the CSV as a byte slice.  The returned slice is only valid until the next
// call to Bytes
func (r *Reader) Bytes() ([]byte, error) {
	r.cell = r.cell[:0]
	var s cellState

	for {
		if r.pos >= len(r.buf) {
			r.buf = r.buf[:cap(r.buf)]
			n, err := r.r.Read(r.buf)
			if n == 0 && err != nil {
				if err == io.EOF {
					r.fileDone = true
					r.rowDone = true
					if s == cellStateInQuote {
						return nil, fmt.Errorf("unexpected EOF")
					}
					return r.cell, nil
				}
				return nil, err
			}
			r.buf = r.buf[:n]
			r.pos = 0
		}

		buf := r.buf[r.pos:]
		for _, c := range buf {
			r.pos++

			switch c {
			case '"':
				// Either enter or exit quotes or something
				switch s {
				case cellStateBegin:
					// This cell is a quoted string
					s = cellStateInQuote
				case cellStateInQuote:
					// Quotes are escaped via two quotes. Or this could be the end of the quote
					s = cellStateInQuoteQuote
				case cellStateInQuoteQuote:
					// Two quotes is an escaped quote
					s = cellStateInQuote
					r.cell = append(r.cell, c)
				case cellStateInCell:
					// just a character once we're in a cell
					r.cell = append(r.cell, c)
				case cellStateTrailingWhiteSpace:
					// TODO: structured errors
					return nil, fmt.Errorf("unexpected quote after quoted string")
				case cellStateSlashR:
					r.cell = append(r.cell, '\r')
					r.cell = append(r.cell, c)
					s = cellStateInCell
				}
			case ',':
				switch s {
				case cellStateInQuote:
					// , inside a quoted cell - just a char
					r.cell = append(r.cell, c)
				case cellStateSlashR:
					r.cell = append(r.cell, '\r')
					fallthrough
				default:
					// end of cell
					s = cellStateBegin
					return r.cell, nil
				}

			case ' ':
				switch s {
				case cellStateBegin, cellStateTrailingWhiteSpace:
					// Skip over initial and trailing white space
				case cellStateInQuote:
					// space inside a quoted cell - just a char
					r.cell = append(r.cell, c)
				case cellStateInQuoteQuote:
					// end of cell, but need to strip trailing white space
					s = cellStateTrailingWhiteSpace
				case cellStateSlashR:
					r.cell = append(r.cell, '\r')
					fallthrough
				case cellStateInCell:
					// TODO: issue with trailing space??
					r.cell = append(r.cell, c)
				}

			case '\r':
				// Need to deal with /r/n for EOF
				switch s {
				case cellStateInQuote, cellStateSlashR:
					// \r inside a quoted cell - just a char
					r.cell = append(r.cell, c)
				default:
					// end of cell
					s = cellStateSlashR
				}

			case '\n':
				switch s {
				case cellStateInQuote:
					// \n inside a quoted cell - just a char
					r.cell = append(r.cell, c)
				default:
					// end of cell
					s = cellStateBegin
					r.rowDone = true
					return r.cell, nil
				}

			default:
				switch s {
				case cellStateBegin:
					s = cellStateInCell
					r.cell = append(r.cell, c)
				case cellStateInQuote:
					// , inside a quoted cell - just a char
					r.cell = append(r.cell, c)
				case cellStateInQuoteQuote:
					// end of cell - but an error
					s = cellStateBegin
					return r.cell, fmt.Errorf("unexpected char %c after terminating quote", c)
				case cellStateSlashR:
					r.cell = append(r.cell, '\r')
					s = cellStateInCell
					fallthrough
				case cellStateInCell:
					r.cell = append(r.cell, c)
				case cellStateTrailingWhiteSpace:
					return nil, fmt.Errorf("unexpected char %c after quoted cell", c)
				}
			}
		}
	}
}
