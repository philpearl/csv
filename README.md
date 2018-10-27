# CSV

[![GoDoc](https://godoc.org/github.com/philpearl/csv?status.svg)](https://godoc.org/github.com/philpearl/csv) 
[![Build Status](https://travis-ci.org/philpearl/csv.svg)](https://travis-ci.org/philpearl/csv)

CSV is a csv writer that doesn't force you to make allocations. The CSV writer in the standard library takes a slice of strings. If your CSV is made up of a whole load of floats, you'll need to create a lot of strings from the floats, and for a big CSV file this can cause a big problem with Garbage.

One day I may also build a reader with similar characteristics.

## Example 
```go
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
```