package csv

import (
	"bytes"
	stdcsv "encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Example() {
	w := NewWriter(os.Stdout)
	// Write a header
	w.String("header1")
	w.String("header2")
	w.String("header3")
	_ = w.LineComplete()
	w.String("cheese")
	w.Float64(1.7)
	w.Bool(true)
	_ = w.LineComplete()
	w.String("carrots")
	w.Float64(3.14)
	w.Bool(false)
	_ = w.LineComplete()
	// Output: header1,header2,header3
	// cheese,1.7,true
	// carrots,3.14,false
}

func TestWriter(t *testing.T) {
	tests := []struct {
		name string
		vals [][]interface{}
		exp  string
	}{
		{
			name: "basic",
			vals: [][]interface{}{
				{"a", "b", "c", "d", "e", "f"},
				{1, "hat", []byte{'a', 'b', 'c'}, 1.73849, false, true},
			},
			exp: "a,b,c,d,e,f\n1,hat,abc,1.73849,false,true\n",
		},
		{
			name: "basic negative",
			vals: [][]interface{}{
				{"a", "b", "c", "d"},
				{1, "hat", []byte{'a', 'b', 'c'}, -1.73849},
			},
			exp: "a,b,c,d\n1,hat,abc,-1.73849\n",
		},
		{
			name: "newline",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat\nlemon", []byte{'a', 'b', '\n', 'c'}},
			},
			exp: "a,b,c\n1,\"hat\nlemon\",\"ab\nc\"\n",
		},
		{
			name: "newline utf8",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat§\n§lemon", []byte{'a', 'b', '\n', 'c'}},
			},
			exp: "a,b,c\n1,\"hat§\n§lemon\",\"ab\nc\"\n",
		},
		{
			name: "comma",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat,lemon", []byte{'a', 'b', 'c', ','}},
			},
			exp: "a,b,c\n1,\"hat,lemon\",\"abc,\"\n",
		},
		{
			name: "double-quote",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat\"lemon", []byte{'a', 'b', 'c', '"'}},
			},
			exp: "a,b,c\n1,\"hat\"\"lemon\",\"abc\"\"\"\n",
		},
		{
			name: "double-quote §",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat§\"§lemon", []byte{'a', 'b', 'c', '"'}},
			},
			exp: "a,b,c\n1,\"hat§\"\"§lemon\",\"abc\"\"\"\n",
		},
		{
			name: "leading space",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, " hatlemon", []byte{' ', 'b', 'c'}},
			},
			exp: "a,b,c\n1,\" hatlemon\",\" bc\"\n",
		},
		{
			name: "internal space",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, "hat lemon", []byte{'a', 'b', 'c'}},
			},
			exp: "a,b,c\n1,hat lemon,abc\n",
		},
		{
			name: "skip",
			vals: [][]interface{}{
				{"a", "b", "c"},
				{1, nil, []byte{'a', 'b', 'c'}},
			},
			exp: "a,b,c\n1,,abc\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b bytes.Buffer
			w := NewWriter(&b)

			for _, line := range test.vals {
				for _, val := range line {
					switch val := val.(type) {
					case string:
						w.String(val)
					case []byte:
						w.Bytes(val)
					case bool:
						w.Bool(val)
					case int64:
						w.Int64(val)
					case int:
						w.Int64(int64(val))
					case float64:
						w.Float64(val)
					case nil:
						w.Skip()
					}
				}
				assert.NoError(t, w.LineComplete())
			}

			assert.Equal(t, test.exp, b.String())
		})
	}
}

func BenchmarkWriter(b *testing.B) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		w.String("hatah")
		for j := 0; j < 100; j++ {
			w.Float64(1.382)
		}
		if err := w.LineComplete(); err != nil {
			b.Fatalf("failed %s", err)
		}
	}
}

func BenchmarkStandardWriter(b *testing.B) {
	var buf bytes.Buffer
	w := stdcsv.NewWriter(&buf)

	b.ResetTimer()
	b.ReportAllocs()
	var line []string
	for i := 0; i < b.N; i++ {
		line = line[:0]
		line = append(line, "hatah")
		for j := 0; j < 100; j++ {
			line = append(line, strconv.FormatFloat(1.382, 'g', -1, 64))
		}
		if err := w.Write(line); err != nil {
			b.Fatalf("failed %s", err)
		}
	}
}
