// Package fixture reads the fixtures internal/oracle writes from Pele: JSON, gzipped, beside
// where the tests name them.
package fixture

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
)

// Read is the JSON of the fixture at path, which is kept as path + ".gz".
func Read(path string) ([]byte, error) {
	data, err := os.ReadFile(path + ".gz")
	if err != nil {
		return nil, err
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}
