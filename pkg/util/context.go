package util

import (
	"context"

	"github.com/sirupsen/logrus"
)

func NewContext(dryRun bool) context.Context {
	return WithDryRun(context.Background(), dryRun)
}

var clusterNumberKey = clusterNumberKeyImpl{}

type clusterNumberKeyImpl struct{}

func WithClusterNumber(ctx context.Context, n uint16) context.Context {
	return context.WithValue(ctx, clusterNumberKey, n)
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

func Logger(ctx context.Context) *logrus.Entry {
	logger := logrus.WithContext(ctx)
	n, ok := ctx.Value(clusterNumberKey).(uint16)
	if ok {
		logger = logger.WithField("cluster", n)
	}
	return logger
}
