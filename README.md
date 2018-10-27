# CSV

[![GoDoc](https://godoc.org/github.com/philpearl/csv?status.svg)](https://godoc.org/github.com/philpearl/csv) 
[![Build Status](https://travis-ci.org/philpearl/csv.svg)](https://travis-ci.org/philpearl/csv)

CSV is a csv writer that doesn't force you to make allocations. The CSV writer in the standard library takes a slice of strings. If your CSV is made up of a whole load of floats, you'll need to create a lot of strings from the floats, and for a big CSV file this can cause a big problem with Garbage.

One day I may also build a reader with similar characteristics.