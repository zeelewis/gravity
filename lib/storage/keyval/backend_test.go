/*
Copyright 2020 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package keyval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompressDecompress(t *testing.T) {
	tt := []struct {
		description string
		size        int
	}{
		{
			description: "smaller than gz header",
			size:        1,
		},
		{
			description: "uncompressed",
			size:        25,
		},
		{
			description: "compress",
			size:        compressAbove + 100,
		},
	}

	for _, test := range tt {
		// test compression
		in := make([]byte, test.size)
		res, err := compress(in)
		assert.NoError(t, err, test.description)

		out, err := decompress(res)
		assert.NoError(t, err, test.description)

		assert.Equal(t, in, out, test.description)
	}
}
