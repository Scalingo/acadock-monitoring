package procfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestLoadAvgReader_Read(t *testing.T) {
	examples := []struct {
		Name    string
		Fixture string
		Expect  LoadAverage
	}{
		{
			Name:    "Simple loadaverage file",
			Fixture: "loadavg_1.txt",
			Expect: LoadAverage{
				Load1:          1.76,
				Load5:          4.08,
				Load10:         4.41,
				RunningProcess: 3,
				TotalProcess:   1484,
				LastPID:        2852530,
			},
		},
	}

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			fs := testFileSystem{
				file: "./fixtures/" + example.Fixture,
			}
			reader := LoadAvgReader{fs: fs}
			res, err := reader.Read(context.Background())
			require.NoError(t, err)
			assert.Equal(t, example.Expect, res)
		})
	}
}
