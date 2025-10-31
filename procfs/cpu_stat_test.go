package procfs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/Scalingo/acadock-monitoring/v2/procfs/procfsmock"

	"github.com/golang/mock/gomock"
)

func TestCPUStatReader_Read(t *testing.T) {
	timeFactor := 10000000
	examples := []struct {
		Name     string
		ClockTck int
		Fixture  string
		Expect   CPUStats
	}{
		{
			Name:     "with simple cpustat file",
			ClockTck: 100,
			Fixture:  "cpustats_1.txt",
			Expect: CPUStats{
				CPUs: map[string]SingleCPUStat{
					"cpu": {
						Name:      "cpu",
						User:      time.Duration(155973 * timeFactor),
						Nice:      time.Duration(111 * timeFactor),
						System:    time.Duration(25140 * timeFactor),
						IDLE:      time.Duration(1665056 * timeFactor),
						IOWait:    time.Duration(1628 * timeFactor),
						IRQ:       time.Duration(5054 * timeFactor),
						SoftIRQ:   time.Duration(1906 * timeFactor),
						Steal:     0,
						Guest:     0,
						GuestNice: 0,
					},
					"cpu0": {
						Name:      "cpu0",
						User:      time.Duration(20172 * timeFactor),
						Nice:      time.Duration(13 * timeFactor),
						System:    time.Duration(3028 * timeFactor),
						IDLE:      time.Duration(207387 * timeFactor),
						IOWait:    time.Duration(220 * timeFactor),
						IRQ:       time.Duration(607 * timeFactor),
						SoftIRQ:   time.Duration(249 * timeFactor),
						Steal:     0,
						Guest:     0,
						GuestNice: 0,
					},
					"cpu1": {
						Name:      "cpu1",
						User:      time.Duration(19340 * timeFactor),
						Nice:      time.Duration(12 * timeFactor),
						System:    time.Duration(3098 * timeFactor),
						IDLE:      time.Duration(208438 * timeFactor),
						IOWait:    time.Duration(212 * timeFactor),
						IRQ:       time.Duration(581 * timeFactor),
						SoftIRQ:   time.Duration(204 * timeFactor),
						Steal:     0,
						Guest:     0,
						GuestNice: 0,
					},
				},
			},
		},
	}

	for _, example := range examples {
		t.Run(example.Name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			fs := testFileSystem{file: "./fixtures/" + example.Fixture}
			sysconf := procfsmock.NewMockSysconf(ctrl)
			sysconf.EXPECT().ClockTick().Return(int64(100), nil)
			reader := CPUStatReader{
				fs:      fs,
				sysconf: sysconf,
			}

			res, err := reader.Read(context.Background())
			require.NoError(t, err)
			assert.Equal(t, example.Expect, res)
		})
	}
}
