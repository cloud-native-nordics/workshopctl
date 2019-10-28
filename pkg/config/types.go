package config

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Domain   string `json:"domain"`
	Clusters uint16 `json:"clusters"`
	GitRepo  string `json:"gitRepo"`
	RootDir  string `json:"-"`

	Provider       string `json:"provider"`
	ServiceAccount string `json:"serviceAccount"`
	CPUs           uint16 `json:"cpus"`
	RAM            uint16 `json:"ram"`
	NodeCount      uint16 `json:"nodeCount"`
}

var _ fmt.Stringer = ClusterNumber(0)

type ClusterNumber uint16

func (n ClusterNumber) String() string {
	return fmt.Sprintf("%02d", n)
}

func ForCluster(n uint16, fn func(ClusterNumber, *logrus.Entry) error) error {
	wg := &sync.WaitGroup{}
	wg.Add(int(n))
	foundErr := false
	for i := ClusterNumber(1); i <= ClusterNumber(n); i++ {
		go func(j ClusterNumber) {
			defer wg.Done()
			entry := logrus.WithField("cluster", j)
			if err := fn(j, entry); err != nil {
				entry.Error(err)
				foundErr = true
			}
		}(i)
	}
	wg.Wait()
	if foundErr {
		return fmt.Errorf("an error occured previously")
	}
	return nil
}
