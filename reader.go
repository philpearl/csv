package csv

import (
	"fmt"
	"io"
	"strconv"
	"unsafe"
)

// Reader is a CSV file reader. Create with NewReader.
type Reader struct {
	r   io.Reader
	buf []byte // Buffer we're reading into
	pos int    // position in buf

	// We copy cell content into parsed as we process it. parsed will contain all the cells of a row one after
	// another. parsed is re-used between rows
	parsed []byte
	// Offsets of cell boundaries within parsed. First offset is always zero
	cellOffsets []int

	// The current row as a slice of []bytes or a slice of strings. These are re-used between rows.
	row  [][]byte
	srow []string

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

// Int reads the i-th cell of the current row as an int. Only valid after a call to Read or Scan.
func (r *Reader) Int(i int) (int, error) {
	b := r.parsed[r.cellOffsets[i]:r.cellOffsets[i+1]]
	return strconv.Atoi(*(*string)(unsafe.Pointer(&b)))
}

// Float reads the i-th cell of the current row as a float. Only valid after a call to Read or Scan.
func (r *Reader) Float(i int) (float64, error) {
	b := r.parsed[r.cellOffsets[i]:r.cellOffsets[i+1]]
	return strconv.ParseFloat(*(*string)(unsafe.Pointer(&b)), 64)
}

// Bool reads the i-th cell of the current row as a boolean value. Only valid after a call to Read or Scan.
func (r *Reader) Bool(i int) (bool, error) {
	b := r.parsed[r.cellOffsets[i]:r.cellOffsets[i+1]]
	return strconv.ParseBool(*(*string)(unsafe.Pointer(&b)))
}

// Text reads the i-th cell of the current row as a string. Only valid after a call to Read or Scan
func (r *Reader) Text(i int) string {
	return r.rowStrings()[i]
}

// Read returns the entire next line of the CSV file as a []string. The slice is only valid until the next
// call to Read, but the underlying strings remain valid.
func (r *Reader) Read() ([]string, error) {
	if err := r.Scan(); err != nil {
		return nil, err
	}

	return r.rowStrings(), nil
}

func (r *Reader) rowStrings() []string {
	if len(r.srow) != 0 {
		return r.srow
	}
	s := string(r.parsed)
	lastOffset := 0
	for _, offset := range r.cellOffsets[1:] {
		r.srow = append(r.srow, s[lastOffset:offset])
		lastOffset = offset
	}
	return r.srow
}

// Bytes returns the next row of the CSV as [][]bytes. This data is only valid until the next call to Bytes (
// or Scan or Read)
func (r *Reader) Bytes() ([][]byte, error) {
	if err := r.Scan(); err != nil {
		return nil, err
	}

	lastOffset := 0
	for _, offset := range r.cellOffsets[1:] {
		r.row = append(r.row, r.parsed[lastOffset:offset])
		lastOffset = offset
	}

	return r.row, nil
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

// Scan reads the next row of the CSV. You can then access cells in the row using Int, Float, Bool or Text.
func (r *Reader) Scan() error {
	if r.fileDone {
		return io.EOF
	}

	r.parsed = r.parsed[:0]
	r.rowDone = false
	r.srow = r.srow[:0]
	r.row = r.row[:0]
	r.cellOffsets = r.cellOffsets[:0]
	r.cellOffsets = append(r.cellOffsets, 0)

	for !r.rowDone {
		if err := r.scanCell(); err != nil {
			return err
		}
		r.cellOffsets = append(r.cellOffsets, len(r.parsed))
	}

	return nil
}

// Len returns the number of cells in the current row. This is valid only after a call to Scan, Bytes or Read
func (r *Reader) Len() int {
	return len(r.cellOffsets) - 1
}

func (r *Reader) scanCell() error {
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
						return io.ErrUnexpectedEOF
					}
					return nil
				}
				return err
			}
			r.buf = r.buf[:n]
			r.pos = 0
		}

		buf := r.buf[r.pos:]
		for _, c := range buf {
			r.pos++

			switch s {
			case cellStateBegin:
				switch c {
				case '"':
					// This cell is a quoted string
					s = cellStateInQuote
				case ',':
					// end of cell
					return nil
				case ' ', '\t':
					// Skip initial white space
				case '\r':
					s = cellStateSlashR
				case '\n':
					// end of cell & row
					r.rowDone = true
					return nil
				default:
					r.parsed = append(r.parsed, c)
					s = cellStateInCell
				}

			case cellStateInCell:
				switch c {
				case ',':
					// end of cell
					return nil
				case '\r':
					s = cellStateSlashR
				case '\n':
					// end of cell & row
					r.rowDone = true
					return nil
				default:
					r.parsed = append(r.parsed, c)
				}

			case cellStateInQuote:
				switch c {
				case '"':
					// Either end of cell, or a quoted quote
					s = cellStateInQuoteQuote
				default:
					r.parsed = append(r.parsed, c)
				}

			case cellStateInQuoteQuote:
				switch c {
				case '"':
					// This cell is a quoted string
					r.parsed = append(r.parsed, c)
					s = cellStateInQuote
				case ',':
					// end of cell
					return nil
				case ' ', '\t':
					s = cellStateTrailingWhiteSpace
				case '\r':
					s = cellStateSlashR
				case '\n':
					// end of cell & row
					r.rowDone = true
					return nil
				default:
					return fmt.Errorf("unexpected char %c after terminating quote", c)
				}

			case cellStateTrailingWhiteSpace:
				switch c {
				case ',':
					// end of cell
					return nil
				case ' ', '\t':
					// skip white space
				case '\r':
					s = cellStateSlashR
				case '\n':
					// end of cell & row
					r.rowDone = true
					return nil
				default:
					return fmt.Errorf("unexpected char %c after quoted cell", c)
				}

			case cellStateSlashR:
				switch c {
				case ',':
					r.parsed = append(r.parsed, '\r')
					return nil
				case '\r':
					r.parsed = append(r.parsed, '\r')
				case '\n':
					// end of cell & row
					r.rowDone = true
					return nil
				default:
					r.parsed = append(r.parsed, '\r', c)
					s = cellStateInCell
				}
			}
		}
	}
}
