package procfs

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	prometheusprocfs "github.com/prometheus/procfs"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

const MountInfoRefreshInterval = 30 * time.Second

type MountInfos interface {
	Mountpoint(major uint64, minor uint64) string
	DevicePath(major uint64, minor uint64) string
}

type MountInfoReader struct {
	getMountInfos func() ([]*prometheusprocfs.MountInfo, error)
	sysDevBlock   string
	dev           string
	devMapper     string

	mutex      *sync.RWMutex
	mountInfos map[string]string
	devices    map[string]string
}

func NewMountInfoReader(ctx context.Context) (*MountInfoReader, error) {
	_, err := prometheusprocfs.NewFS("/proc")
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create procfs filesystem")
	}
	pid := os.Getpid()

	return &MountInfoReader{
		getMountInfos: func() ([]*prometheusprocfs.MountInfo, error) {
			return prometheusprocfs.GetProcMounts(pid)
		},
		sysDevBlock: "/sys/dev/block",
		dev:         "/dev",
		devMapper:   "/dev/mapper",
		mutex:       &sync.RWMutex{},
		mountInfos:  make(map[string]string),
		devices:     make(map[string]string),
	}, nil
}

func (r *MountInfoReader) Start(ctx context.Context) {
	log := logger.Get(ctx)

	err := r.Refresh(ctx)
	if err != nil {
		log.WithError(err).Error("Refresh mount infos")
	}

	tick := time.NewTicker(MountInfoRefreshInterval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info("Mount info refresh stopped - Context done")
			return
		case <-tick.C:
			err := r.Refresh(ctx)
			if err != nil {
				log.WithError(err).Error("Refresh mount infos")
			}
		}
	}
}

func (r *MountInfoReader) Refresh(ctx context.Context) error {
	mountInfos, err := r.getMountInfos()
	if err != nil {
		return errors.Wrap(ctx, err, "get mount infos")
	}

	mapperPaths, err := r.mapperPaths(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get device mapper paths")
	}
	devices, err := r.devicePaths(ctx, mapperPaths)
	if err != nil {
		return errors.Wrap(ctx, err, "get device paths")
	}

	infos := make(map[string]string, len(mountInfos))
	for _, mountInfo := range mountInfos {
		infos[mountInfo.MajorMinorVer] = mountInfo.MountPoint
	}

	r.mutex.Lock()
	r.mountInfos = infos
	r.devices = devices
	r.mutex.Unlock()

	return nil
}

func (r *MountInfoReader) Mountpoint(major uint64, minor uint64) string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.mountInfos[deviceKey(major, minor)]
}

func (r *MountInfoReader) DevicePath(major uint64, minor uint64) string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.devices[deviceKey(major, minor)]
}

func (r *MountInfoReader) devicePaths(ctx context.Context, mapperPaths map[string]string) (map[string]string, error) {
	entries, err := os.ReadDir(r.sysDevBlock)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "read sysfs device block directory")
	}

	devices := make(map[string]string, len(entries))
	for _, entry := range entries {
		path, err := r.devicePath(ctx, entry.Name(), mapperPaths)
		if err != nil {
			return nil, errors.Wrapf(ctx, err, "get device path for %s", entry.Name())
		}
		devices[entry.Name()] = path
	}

	return devices, nil
}

func (r *MountInfoReader) devicePath(ctx context.Context, key string, mapperPaths map[string]string) (string, error) {
	target, err := filepath.EvalSymlinks(filepath.Join(r.sysDevBlock, key))
	if err != nil {
		return "", errors.Wrap(ctx, err, "resolve sysfs device symlink")
	}
	devicePath := filepath.Join(r.dev, filepath.Base(target))
	if mapperPath := mapperPaths[devicePath]; mapperPath != "" {
		return mapperPath, nil
	}

	return devicePath, nil
}

func (r *MountInfoReader) mapperPaths(ctx context.Context) (map[string]string, error) {
	entries, err := os.ReadDir(r.devMapper)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "read device mapper directory")
	}

	paths := make(map[string]string, len(entries))
	for _, entry := range entries {
		path := filepath.Join(r.devMapper, entry.Name())
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, errors.Wrapf(ctx, err, "resolve device mapper symlink %s", path)
		}
		paths[target] = path
	}

	return paths, nil
}

func deviceKey(major uint64, minor uint64) string {
	return strconv.FormatUint(major, 10) + ":" + strconv.FormatUint(minor, 10)
}
