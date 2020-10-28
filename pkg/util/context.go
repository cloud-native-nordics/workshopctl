package util

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

func NewContext(dryRun bool, rootPath string) context.Context {
	ctx := context.Background()
	ctx = WithDryRun(ctx, dryRun)
	ctx = withRootPath(ctx, rootPath)
	return ctx
}

var clusterNumberKey = clusterNumberKeyImpl{}

type clusterNumberKeyImpl struct{}

func WithClusterNumber(ctx context.Context, n uint16) context.Context {
	return context.WithValue(ctx, clusterNumberKey, n)
}

func getClusterNumber(ctx context.Context) (uint16, bool) {
	n, ok := ctx.Value(clusterNumberKey).(uint16)
	if !ok {
		logrus.Debug("Didn't find cluster number from context")
	}
	return n, ok
}

var dryRunKey = dryRunKeyImpl{}

type dryRunKeyImpl struct{}

func WithDryRun(ctx context.Context, dryRun bool) context.Context {
	return context.WithValue(ctx, dryRunKey, dryRun)
}

func IsDryRun(ctx context.Context) bool {
	dryRun, ok := ctx.Value(dryRunKey).(bool)
	if !ok {
		logrus.Warn("Expected to be able to get dry-run from context, but got nothing")
		logrus.Warn("Setting dryRun to be true because of this")
		dryRun = true
	}
	return dryRun
}

var rootPathKey = rootPathKeyImpl{}

type rootPathKeyImpl struct{}

func withRootPath(ctx context.Context, rootPath string) context.Context {
	// Always make the path absolute before putting it into the context
	if !filepath.IsAbs(rootPath) {
		wd, err := os.Getwd()
		if err != nil {
			logrus.Fatalf("Failed to get working directory due to: %v. Either fix the underlying problem or specify an absolute path to --root-dir", err)
		}
		rootPath = filepath.Join(wd, rootPath)
	}
	return context.WithValue(ctx, rootPathKey, rootPath)
}

func getRootPath(ctx context.Context) (string, bool) {
	rootPath, ok := ctx.Value(rootPathKey).(string)
	if !ok {
		logrus.Debug("Didn't find rootPath from context, defaulting to '.'")
		return ".", false
	}
	return rootPath, ok
}

// If called without any filePaths this just returns the RootPath
func JoinPaths(ctx context.Context, filePaths ...string) string {
	rootPath, _ := getRootPath(ctx)
	filePaths = append([]string{rootPath}, filePaths...)
	return filepath.Join(filePaths...)
}

var muxKey = muxKeyImpl{}

type muxKeyImpl struct{}

func WithMutex(ctx context.Context, mux *sync.Mutex) context.Context {
	return context.WithValue(ctx, muxKey, mux)
}

func GetMutex(ctx context.Context) (*sync.Mutex, bool) {
	mux, ok := ctx.Value(muxKey).(*sync.Mutex)
	if !ok {
		logrus.Debug("Didn't find mux from context, defaulting to nil")
		return nil, false
	}
	return mux, ok
}

func Logger(ctx context.Context) *logrus.Entry {
	logger := logrus.WithContext(ctx)
	// If cluster number is set, add that logging field
	if n, ok := getClusterNumber(ctx); ok {
		logger = logger.WithField("cluster", n)
	}
	// If root path is set on ctx and debug logging is enabled, add the root-path field
	if rootPath, ok := getRootPath(ctx); ok && logrus.IsLevelEnabled(logrus.DebugLevel) {
		logger = logger.WithField("root-path", rootPath)
	}
	return logger
}
