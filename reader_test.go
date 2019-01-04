package csv_test

import (
	"bytes"
	csvstd "encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"testing"

	"github.com/philpearl/csv"
	"github.com/stretchr/testify/assert"
)

type repeatReader struct {
	content []byte
	read    int
}

func (r *repeatReader) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		m := copy(buf[n:], r.content[r.read:])
		n += m
		r.read += m

		if r.read >= len(r.content) {
			r.read = 0
		}
	}
	return n, nil
}

func TestRepeatReader(t *testing.T) {
	r := repeatReader{content: []byte{1, 2, 3, 4, 5}}

	buf := make([]byte, 9)

	n, err := r.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 1, 2, 3, 4}, buf)
}

func ExampleReader() {
	buf := bytes.NewReader([]byte(`string,int, float
hat, 37, 12.4
Bionic, 12, 97.823`))

	r := csv.NewReader(buf)

	headings, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(headings)

	for {
		err := r.Scan()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		s := r.Text(0)

		i, err := r.Int(1)
		if err != nil {
			log.Fatal(err)
		}

		f, err := r.Float(2)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("s=%s, i=%d, f=%f\n", s, i, f)
	}

	// output: [string int float]
	// s=hat, i=37, f=12.400000
	// s=Bionic, i=12, f=97.823000
}

func TestReadBool(t *testing.T) {
	in := bytes.NewBufferString(`false, true, cheese`)
	r := csv.NewReader(in)

	err := r.Scan()
	assert.NoError(t, err)

	b, err := r.Bool(0)
	assert.NoError(t, err)
	assert.False(t, b)

	b, err = r.Bool(1)
	assert.NoError(t, err)
	assert.True(t, b)

	_, err = r.Bool(2)
	assert.EqualError(t, err, "strconv.ParseBool: parsing \"cheese\": invalid syntax")

	assert.Equal(t, 3, r.Len())
}

func TestReadInt(t *testing.T) {
	in := bytes.NewBufferString(`1, 42, 13.2`)
	r := csv.NewReader(in)

	err := r.Scan()
	assert.NoError(t, err)

	i, err := r.Int(0)
	assert.NoError(t, err)
	assert.Equal(t, 1, i)

	i, err = r.Int(1)
	assert.NoError(t, err)
	assert.Equal(t, 42, i)

	_, err = r.Int(2)
	assert.EqualError(t, err, "strconv.Atoi: parsing \"13.2\": invalid syntax")
}

func TestReadFloat(t *testing.T) {
	in := bytes.NewBufferString(`1, 42.2, 12h2`)
	r := csv.NewReader(in)
	err := r.Scan()
	assert.NoError(t, err)

	f, err := r.Float(0)
	assert.NoError(t, err)
	assert.Equal(t, 1.0, f)

	f, err = r.Float(1)
	assert.NoError(t, err)
	assert.Equal(t, 42.2, f)

	_, err = r.Float(2)
	assert.EqualError(t, err, "strconv.ParseFloat: parsing \"12h2\": invalid syntax")
}

func TestRead(t *testing.T) {

	tests := []struct {
		name string
		in   string
		exp  [][]string
		err  string
	}{
		{
			name: "some basics",
			in: `a,b,c
1, 2, 3
"4", "5,6", "7"`,
			exp: [][]string{
				[]string{"a", "b", "c"},
				[]string{"1", "2", "3"},
				[]string{"4", "5,6", "7"},
			},
		},
		{
			name: "quote special cases",
			in: `"a
 hat","b,,,","c "" hat",`,
			exp: [][]string{
				[]string{"a\n hat", "b,,,", "c \" hat", ""},
			},
		},
		{
			name: "quote in string",
			in:   `a"b`,
			exp: [][]string{
				[]string{"a\"b"},
			},
		},
		{
			name: "white space ",
			in:   `  "bc"  , a z, d`,
			exp: [][]string{
				[]string{"bc", "a z", "d"},
			},
		},
		{
			name: "empty",
			in: `,,"","",,
`,
			exp: [][]string{
				[]string{"", "", "", "", "", ""},
				[]string{""},
			},
		},
		{
			name: "EOF in quote",
			in:   `"`,
			err:  "unexpected EOF",
			exp:  [][]string(nil),
		},
		{
			name: "quote after quote",
			in:   `"a" "`,
			err:  "unexpected char \" after quoted cell",
			exp: [][]string{
				[]string{""},
			},
		},
		{
			name: "char after quote",
			in:   `"a"b`,
			err:  "unexpected char b after terminating quote",
			exp: [][]string{
				[]string{""},
			},
		},
		{
			name: "char after quote space",
			in:   `"a" b`,
			err:  "unexpected char b after quoted cell",
			exp: [][]string{
				[]string{""},
			},
		},
		{
			name: "\\r\\n",
			in:   "a, b\r\nc,d",
			exp: [][]string{
				[]string{"a", "b"},
				[]string{"c", "d"},
			},
		},
		{
			name: "\\r\\r\\n",
			in:   "a, b\r\r\nc,d",
			exp: [][]string{
				[]string{"a", "b\r"},
				[]string{"c", "d"},
			},
		},

		{
			name: "\\rn",
			in:   "a, b\rnc,d",
			exp: [][]string{
				[]string{"a", "b\rnc", "d"},
			},
		},

		{
			name: "\\r\\n in quote",
			in:   "\"b\r\nc\",d",
			exp: [][]string{
				[]string{"b\r\nc", "d"},
			},
		},

		{
			name: "\\r space",
			in:   "b\r c,d",
			exp: [][]string{
				[]string{"b\r c", "d"},
			},
		},
		{
			name: "\\r,",
			in:   "b\r,d",
			exp: [][]string{
				[]string{"b\r", "d"},
			},
		},
		{
			name: "\\r\"",
			in:   "b\r\",d",
			exp: [][]string{
				[]string{"b\r\"", "d"},
			},
		},
	}

	var r *csv.Reader

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			in := bytes.NewReader([]byte(test.in))
			if r == nil {
				r = csv.NewReader(in)
			} else {
				r.SetInput(in)
			}

			var actual [][]string
			for {
				ss, err := r.Read()

				if err != nil {
					if err == io.EOF {
						break
					} else {
						assert.EqualError(t, err, test.err)
					}
				} else {
					row := make([]string, len(ss))
					copy(row, ss)
					actual = append(actual, row)
				}
			}

			assert.Equal(t, test.exp, actual)
		})
	}
}

func BenchmarkRead(b *testing.B) {
	content := []byte(`cheese, feet, lemon, 99, 1002, 1298, 12.3, 17, 11, whale
`)
	buf := &repeatReader{content: content}

	r := csv.NewReader(buf)

	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	total := 0.0
	for i := 0; i < b.N; i++ {
		if err := r.Scan(); err != nil {
			b.Fatal(err)
		}
		f, err := r.Float(6)
		if err != nil {
			b.Fatal(err)
		}
		total += f
	}
	assert.InEpsilon(b, float64(b.N)*12.3, total, 0.1)
}

func BenchmarkReadStdlib(b *testing.B) {
	content := []byte(`cheese, feet, lemon, 99, 1002, 1298, 12.3, 17, 11, whale
`)
	buf := &repeatReader{content: content}

	r := csvstd.NewReader(buf)
	r.ReuseRecord = true

	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	total := 0.0
	for i := 0; i < b.N; i++ {
		cells, err := r.Read()
		if err != nil {
			b.Fatal(err)
		}

		f, err := strconv.ParseFloat(strings.TrimSpace(cells[6]), 64)
		if err != nil {
			b.Fatal(err)
		}
		total += f
	}

	assert.InEpsilon(b, float64(b.N)*12.3, total, 0.1)
}

func BenchmarkReadOldWay(b *testing.B) {
	// Using this CSV library in the same way as the standard one is a little faster, probably because this
	// one is not flexible about separators, etc
	content := []byte(`cheese, feet, lemon, 99, 1002, 1298, 12.3, 17, 11, whale
`)
	buf := &repeatReader{content: content}

	r := csv.NewReader(buf)

	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	total := 0.0
	for i := 0; i < b.N; i++ {
		cells, err := r.Read()
		if err != nil {
			b.Fatal(err)
		}

		f, err := strconv.ParseFloat(cells[6], 64)
		if err != nil {
			b.Fatal(err)
		}
		total += f
	}

	assert.InEpsilon(b, float64(b.N)*12.3, total, 0.1)
}
