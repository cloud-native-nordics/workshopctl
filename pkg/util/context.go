package util

import (
	"context"

	"github.com/sirupsen/logrus"
)

var clusterNumberKey = clusterNumberKeyImpl{}

type clusterNumberKeyImpl struct{}

func WithClusterNumber(ctx context.Context, n uint16) context.Context {
	return context.WithValue(ctx, clusterNumberKey, n)
}

func Logger(ctx context.Context) *logrus.Entry {
	logger := logrus.WithContext(ctx)
	n, ok := ctx.Value(clusterNumberKey).(uint16)
	if ok {
		logger = logger.WithField("cluster", n)
	}
	return logger
}
