package csv

import (
	"fmt"
	"io"
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
		buf: make([]byte, 0, 4096),
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

// Bytes returns the next cell in the CSV as a byte slice.  The returned slice is only valid until the next
// call to Bytes
func (r *Reader) Bytes() ([]byte, error) {
	r.cell = r.cell[:0]
	var s cellState

	for {
		c, err := r.next()
		// fmt.Printf("%d(%c) in %s (%v)\n", c, c, r.s, err)
		if err != nil {
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
			}
		case ',':
			switch s {
			case cellStateInQuote:
				// , inside a quoted cell - just a char
				r.cell = append(r.cell, c)
			default:
				// end of cell
				s = cellStateBegin
				return r.cell, nil
			}

		case ' ':
			switch s {
			case cellStateBegin, cellStateTrailingWhiteSpace:
				// Skip over initial white space
			case cellStateInQuote:
				// space inside a quoted cell - just a char
				r.cell = append(r.cell, c)
			case cellStateInQuoteQuote:
				// end of cell, but need to strip trailing white space
				s = cellStateTrailingWhiteSpace
			case cellStateInCell:
				// TODO: issue with trailing space??
				r.cell = append(r.cell, c)
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
			case cellStateInCell:
				r.cell = append(r.cell, c)
			case cellStateTrailingWhiteSpace:
				return nil, fmt.Errorf("unexpected char %c after quoted cell", c)
			}
		}
	}
}

func (r *Reader) next() (byte, error) {
	if r.pos >= len(r.buf) {
		r.buf = r.buf[:cap(r.buf)]
		n, err := r.r.Read(r.buf)
		if n == 0 && err != nil {
			return 0, err
		}
		r.buf = r.buf[:n]
		r.pos = 0
	}

	c := r.buf[r.pos]
	r.pos++
	return c, nil
}
