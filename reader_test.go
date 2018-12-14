package csv_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/philpearl/csv"
	"github.com/stretchr/testify/assert"
)

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
			exp: [][]string{
				[]string(nil),
			},
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			in := bytes.NewReader([]byte(test.in))
			r := csv.NewReader(in)

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
				actual = append(actual, row)
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
