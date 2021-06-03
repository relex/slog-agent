package hybridbuffer

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/xattr"
	"github.com/relex/gotils/logger"
	"github.com/relex/slog-agent/base"
	"github.com/relex/slog-agent/util"
	"golang.org/x/sys/unix"
)

const queueDirHashLength = 8
const xattrBufferID = "user.hybridbufferID"

func makeBufferQueueDir(parentLogger logger.Logger, rootPath string, bufferID string) string {
	var path string
	if bufferID != "" {
		dirname := sanitizeDirName(bufferID)
		if dirname != bufferID {
			parentLogger.Warnf("unclean buffer ID as dirname: '%s'", bufferID)
		}
		// if buffer ID is not the same after sanitization, it would still get unique dir due to hash
		hash := util.MD5ToHexdigest(bufferID)
		path = filepath.Join(rootPath, dirname+"."+hash[len(hash)-queueDirHashLength:])
	} else {
		path = rootPath
	}
	if derr := os.MkdirAll(path, 0755); derr != nil {
		parentLogger.Errorf("error creating queue dir path='%s': %s", path, derr.Error())
	}
	if xerr := xattr.Set(path, xattrBufferID, []byte(bufferID)); xerr != nil {
		parentLogger.Errorf("error labelling id on queue dir path='%s': %s", path, xerr)
	}
	return path
}

func listBufferQueueIDs(parentLogger logger.Logger, rootPath string, matchChunkID func(string) bool,
	parentMetricFactory *base.MetricFactory) []string {

	metricFactory := makeBufferMetricsFactory(parentMetricFactory)

	parentLogger.Infof("scan root dir: %s", rootPath)
	rootDir, oerr := os.Open(rootPath)
	if oerr != nil {
		if os.IsNotExist(oerr) {
			return nil
		}
		parentLogger.Errorf("error opening root dir: %s", oerr.Error())
		return nil
	}
	defer rootDir.Close()

	entryNames, rerr := rootDir.Readdirnames(0)
	if rerr != nil {
		parentLogger.Errorf("error scanning root dir: %s", rerr.Error())
		return nil
	}
	sort.Strings(entryNames)

	validBufferIDList := make([]string, 0, len(entryNames))
	for _, name := range entryNames {
		path := filepath.Join(rootPath, name)

		// check file entry is dir
		stat, serr := util.StatFileAt(rootDir, name)
		if serr != nil {
			parentLogger.Errorf("error stating entry path='%s': %s", path, serr.Error())
			continue
		}
		if stat.Mode&unix.DT_DIR == 0 {
			continue
		}

		// read ID from extended attribute
		idBytes, xerr := xattr.Get(path, xattrBufferID)
		if xerr != nil {
			parentLogger.Warnf("ignore buffer dir without id, path='%s': %s", path, xerr.Error())
			continue
		}
		if len(idBytes) == 0 {
			parentLogger.Warnf("ignore buffer dir with empty id, path='%s': %s", path, xerr.Error())
			continue
		}
		id := util.StringFromBytes(idBytes)

		// count chunks in the dir
		op := newChunkOperator(parentLogger, path, matchChunkID, metricFactory, 0)
		if numChunks := op.CountExistingChunks(); numChunks > 0 {
			validBufferIDList = append(validBufferIDList, id)
			parentLogger.Infof("add existing buffer name='%s' id='%s' count=%d", name, id, numChunks)
		} else {
			parentLogger.Infof("skip empty buffer name='%s' id='%s'", name, id)
		}
		op.Close()
	}
	return validBufferIDList
}

func sanitizeDirName(name string) string {
	result := make([]byte, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch c {
		case 0, '/':
			c = '_'
		}
		result[i] = c
	}
	return util.StringFromBytes(result)
}
