package csv_test

import (
	"bytes"
	csvstd "encoding/csv"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/philpearl/csv"
	"github.com/stretchr/testify/assert"
)

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

	for r.Scan() {
		s, err := r.Text()
		if err != nil {
			log.Fatal(err)
		}

		i, err := r.Int()
		if err != nil {
			log.Fatal(err)
		}

		f, err := r.Float()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("s=%s, i=%d, f=%f\n", s, i, f)
	}

	// output: [string int float]
	// s=hat, i=37, f=12.400000
	// s=Bionic, i=12, f=97.823000
}

func ExampleReader_ScanLine() {
	buf := bytes.NewReader([]byte(`string,int, float
hat, 37, 12.4
Bionic, 12, 97.823`))

	r := csv.NewReader(buf)

	headings, err := r.Read()
	if err != nil {
		log.Fatal(err)
	}

	// Work out which is the "int" column
	var indexInt int
	for i, h := range headings {
		if h == "int" {
			indexInt = i
			break
		}
	}

	for r.Scan() {
		for index := 0; r.ScanLine(); index++ {
			switch index {
			case indexInt:
				// This is the int column
				i, err := r.Int()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("i=%d\n", i)
			default:
				// Skip this cell
				_, err = r.Bytes()
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	// output: i=37
	// i=12
}

func TestReadBool(t *testing.T) {
	in := bytes.NewBufferString(`false, true, cheese`)
	r := csv.NewReader(in)

	b, err := r.Bool()
	assert.NoError(t, err)
	assert.False(t, b)

	b, err = r.Bool()
	assert.NoError(t, err)
	assert.True(t, b)

	_, err = r.Bool()
	assert.EqualError(t, err, "strconv.ParseBool: parsing \"cheese\": invalid syntax")

	_, err = r.Bool()
	assert.Equal(t, csv.ErrRowDone, err)
}

func TestReadInt(t *testing.T) {
	in := bytes.NewBufferString(`1, 42, 13.2`)
	r := csv.NewReader(in)

	i, err := r.Int()
	assert.NoError(t, err)
	assert.Equal(t, 1, i)

	i, err = r.Int()
	assert.NoError(t, err)
	assert.Equal(t, 42, i)

	_, err = r.Int()
	assert.EqualError(t, err, "strconv.Atoi: parsing \"13.2\": invalid syntax")

	_, err = r.Int()
	assert.Equal(t, csv.ErrRowDone, err)
}

func TestReadFloat(t *testing.T) {
	in := bytes.NewBufferString(`1, 42.2, 12h2`)
	r := csv.NewReader(in)

	f, err := r.Float()
	assert.NoError(t, err)
	assert.Equal(t, 1.0, f)

	f, err = r.Float()
	assert.NoError(t, err)
	assert.Equal(t, 42.2, f)

	_, err = r.Float()
	assert.EqualError(t, err, "strconv.ParseFloat: parsing \"12h2\": invalid syntax")

	_, err = r.Float()
	assert.Equal(t, csv.ErrRowDone, err)
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
			err:  "unexpected quote after quoted string",
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
			for r.Scan() {
				var row []string
				for r.ScanLine() {
					b, err := r.Bytes()
					if err != nil {
						assert.EqualError(t, err, test.err)
					} else {
						row = append(row, string(b))
					}
				}
				if row != nil {
					actual = append(actual, row)
				}
			}

			assert.Equal(t, test.exp, actual)
		})
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			in := bytes.NewReader([]byte(test.in))
			r := csv.NewReader(in)

			var actual [][]string
			for r.Scan() {
				row, err := r.Read()
				if err != nil {
					assert.EqualError(t, err, test.err)
				} else {
					actual = append(actual, row)
				}
			}
			assert.Equal(t, test.exp, actual)
		})
	}

}

func BenchmarkRead(b *testing.B) {

	buf := bytes.NewReader([]byte(`a,b,c,d,efgdh
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99`))

	r := csv.NewReader(buf)

	b.SetBytes(int64(buf.Len()))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Seek(0, io.SeekStart)
		r.SetInput(buf)

		count := 0
		for r.Scan() {
			for r.ScanLine() {
				c, err := r.Bytes()
				if err != nil {
					b.Fatal(err)
				}
				count += len(c)
			}
		}
		if count != 359 {
			b.Fatalf("read %d bytes", count)
		}
	}
}

func BenchmarkReadFloats(b *testing.B) {

	buf := bytes.NewReader([]byte(`1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14
1, 2, 3, 4, 5, 6, 78, 8.23, 1e14`))

	r := csv.NewReader(buf)

	b.SetBytes(int64(buf.Len()))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Seek(0, io.SeekStart)
		r.SetInput(buf)

		total := 0.0
		for r.Scan() {
			for r.ScanLine() {
				c, err := r.Float()
				if err != nil {
					b.Fatal(err)
				}
				total += c
			}
		}
		if total != 1100000000001179.750000 {
			b.Fatalf("total %f", total)
		}
	}
}

func BenchmarkReadStdlib(b *testing.B) {

	buf := bytes.NewReader([]byte(`a,b,c,d
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99
"abcdefg""hi", zzpza, §§§§, 99`))

	b.SetBytes(int64(buf.Len()))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Seek(0, io.SeekStart)
		r := csvstd.NewReader(buf)
		r.ReuseRecord = true

		count := 0

		for {
			c, err := r.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				b.Fatal(err)
			}
			count += len(c)

		}
		if count != 60 {
			b.Fatalf("read %d bytes", count)
		}
	}
}
