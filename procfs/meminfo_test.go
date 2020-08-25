package procfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemInfoReader_Read(t *testing.T) {
	scaleFactor := 1024
	examples := []struct {
		Name    string
		Fixture string
		Expect  MemInfo
	}{
		{
			Name:    "with a simple meminfo",
			Fixture: "meminfo_1.txt",
			Expect: MemInfo{
				MemTotal:     uint64(32457876 * scaleFactor),
				MemFree:      uint64(22619848 * scaleFactor),
				Buffers:      uint64(2824 * scaleFactor),
				Cached:       uint64(5205948 * scaleFactor),
				SwapCached:   uint64(0),
				Active:       uint64(5511916 * scaleFactor),
				Inactive:     uint64(2588232 * scaleFactor),
				ActiveAnon:   uint64(3780368 * scaleFactor),
				InactiveAnon: uint64(382512 * scaleFactor),
				ActiveFile:   uint64(1731548 * scaleFactor),
				InactiveFile: uint64(2205720 * scaleFactor),
				Unevictable:  uint64(893768 * scaleFactor),
				MLocked:      uint64(32 * scaleFactor),
				SwapTotal:    uint64(33554428 * scaleFactor),
				SwapFree:     uint64(33554428 * scaleFactor),
				Dirty:        uint64(284 * scaleFactor),
				Writeback:    uint64(0),
				AnonPages:    uint64(3785184 * scaleFactor),
				Mapped:       uint64(936140 * scaleFactor),
				Slab:         uint64(529640 * scaleFactor),
				SReclaimable: uint64(310756 * scaleFactor),
				SUnreclaim:   uint64(218884 * scaleFactor),
				PurgeTables:  uint64(0),
				NFSUnstable:  uint64(0),
				Bounce:       uint64(0),
				WritebackTmp: uint64(0),
				CommitLimit:  uint64(49783364 * scaleFactor),
				CommittedAS:  uint64(11684876 * scaleFactor),
				VmallocTotal: uint64(34359738367 * scaleFactor),
				VmallocUsed:  uint64(43620 * scaleFactor),
				VmallocChunk: uint64(0),
				DirectMap4k:  uint64(309316 * scaleFactor),
				DirectMap2M:  uint64(9744384 * scaleFactor),
				DirectMap1G:  uint64(24117248 * scaleFactor),
			},
		},
	}

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			fs := testFileSystem{file: "./fixtures/" + example.Fixture}
			reader := MemInfoReader{fs: fs}
			res, err := reader.Read(context.Background())
			require.NoError(t, err)
			assert.Equal(t, example.Expect, res)
		})
	}
}
