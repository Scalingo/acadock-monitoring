package procfs

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	prometheusprocfs "github.com/prometheus/procfs"
	"github.com/stretchr/testify/require"
)

func TestMountInfoReaderRefreshStoresMountPointsByDevice(t *testing.T) {
	tmp := t.TempDir()
	sysDevBlock := filepath.Join(tmp, "sys", "dev", "block")
	dev := filepath.Join(tmp, "dev")
	devMapper := filepath.Join(dev, "mapper")
	deviceTarget1 := filepath.Join(tmp, "sys", "devices", "pci0000:00", "block", "sda", "sda1")
	deviceTarget2 := filepath.Join(tmp, "sys", "devices", "pci0000:00", "block", "sda", "sda2")
	deviceTarget3 := filepath.Join(tmp, "sys", "devices", "pci0000:00", "block", "sdb")
	require.NoError(t, os.MkdirAll(sysDevBlock, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(deviceTarget1), 0o755))
	require.NoError(t, os.MkdirAll(devMapper, 0o755))
	require.NoError(t, os.WriteFile(deviceTarget1, nil, 0o644))
	require.NoError(t, os.WriteFile(deviceTarget2, nil, 0o644))
	require.NoError(t, os.WriteFile(deviceTarget3, nil, 0o644))
	require.NoError(t, os.Symlink(deviceTarget1, filepath.Join(sysDevBlock, "8:1")))
	require.NoError(t, os.Symlink(deviceTarget2, filepath.Join(sysDevBlock, "8:2")))
	require.NoError(t, os.Symlink(deviceTarget3, filepath.Join(sysDevBlock, "8:16")))
	require.NoError(t, os.WriteFile(filepath.Join(dev, "sda1"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dev, "sda2"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dev, "sdb"), nil, 0o644))

	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return []*prometheusprocfs.MountInfo{
				{MajorMinorVer: "8:1", MountPoint: "/"},
				{MajorMinorVer: "8:2", MountPoint: "/var/lib"},
			}, nil
		},
		sysDevBlock: sysDevBlock,
		dev:         dev,
		devMapper:   devMapper,
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.NoError(t, err)
	require.Equal(t, "/", reader.Mountpoint(8, 1))
	require.Equal(t, filepath.Join(dev, "sda1"), reader.DevicePath(8, 1))
	require.Equal(t, "/var/lib", reader.Mountpoint(8, 2))
	require.Equal(t, filepath.Join(dev, "sda2"), reader.DevicePath(8, 2))
	require.Empty(t, reader.Mountpoint(8, 16))
	require.Equal(t, filepath.Join(dev, "sdb"), reader.DevicePath(8, 16))
	require.Empty(t, reader.Mountpoint(8, 3))
}

func TestMountInfoReaderRefreshPrefersMapperAlias(t *testing.T) {
	tmp := t.TempDir()
	sysDevBlock := filepath.Join(tmp, "sys", "dev", "block")
	dev := filepath.Join(tmp, "dev")
	devMapper := filepath.Join(dev, "mapper")
	deviceTarget := filepath.Join(tmp, "sys", "devices", "virtual", "block", "dm-0")
	require.NoError(t, os.MkdirAll(sysDevBlock, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(deviceTarget), 0o755))
	require.NoError(t, os.MkdirAll(devMapper, 0o755))
	require.NoError(t, os.WriteFile(deviceTarget, nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dev, "dm-0"), nil, 0o644))
	require.NoError(t, os.Symlink(deviceTarget, filepath.Join(sysDevBlock, "253:0")))
	require.NoError(t, os.Symlink(filepath.Join("..", "dm-0"), filepath.Join(devMapper, "data")))

	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return []*prometheusprocfs.MountInfo{{MajorMinorVer: "253:0", MountPoint: "/data"}}, nil
		},
		sysDevBlock: sysDevBlock,
		dev:         dev,
		devMapper:   devMapper,
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(devMapper, "data"), reader.DevicePath(253, 0))
}

func TestMountInfoReaderRefreshReturnsError(t *testing.T) {
	expectedErr := errors.New("boom")
	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return nil, expectedErr
		},
		mutex:      &sync.RWMutex{},
		mountInfos: make(map[string]string),
		devices:    make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.ErrorIs(t, err, expectedErr)
}

func TestMountInfoReaderRefreshReturnsMapperPathError(t *testing.T) {
	tmp := t.TempDir()
	sysDevBlock := filepath.Join(tmp, "sys", "dev", "block")
	dev := filepath.Join(tmp, "dev")
	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return []*prometheusprocfs.MountInfo{{MajorMinorVer: "8:1", MountPoint: "/"}}, nil
		},
		sysDevBlock: sysDevBlock,
		dev:         dev,
		devMapper:   filepath.Join(dev, "mapper"),
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.ErrorContains(t, err, "read device mapper directory")
}

func TestMountInfoReaderRefreshReturnsDevicePathError(t *testing.T) {
	tmp := t.TempDir()
	sysDevBlock := filepath.Join(tmp, "sys", "dev", "block")
	dev := filepath.Join(tmp, "dev")
	devMapper := filepath.Join(dev, "mapper")
	require.NoError(t, os.MkdirAll(devMapper, 0o755))
	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return []*prometheusprocfs.MountInfo{{MajorMinorVer: "8:1", MountPoint: "/"}}, nil
		},
		sysDevBlock: sysDevBlock,
		dev:         dev,
		devMapper:   devMapper,
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.ErrorContains(t, err, "read sysfs device block directory")
}

func TestMountInfoReaderRefreshReturnsDeviceSymlinkError(t *testing.T) {
	tmp := t.TempDir()
	sysDevBlock := filepath.Join(tmp, "sys", "dev", "block")
	dev := filepath.Join(tmp, "dev")
	devMapper := filepath.Join(dev, "mapper")
	require.NoError(t, os.MkdirAll(sysDevBlock, 0o755))
	require.NoError(t, os.MkdirAll(devMapper, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(tmp, "missing"), filepath.Join(sysDevBlock, "8:1")))
	reader := MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return nil, nil
		},
		sysDevBlock: sysDevBlock,
		dev:         dev,
		devMapper:   devMapper,
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}

	err := reader.Refresh(t.Context())
	require.ErrorContains(t, err, "resolve sysfs device symlink")
}
