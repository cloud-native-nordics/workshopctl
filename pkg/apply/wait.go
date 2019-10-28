package apply

import (
	"fmt"
	"net"
	"time"

	"github.com/luxas/workshopctl/pkg/config"
	"github.com/luxas/workshopctl/pkg/util"
)

type Waiter struct {
	*config.ClusterInfo
}

func NewWaiter(info *config.ClusterInfo) *Waiter {
	return &Waiter{info}
}

func (w *Waiter) execKubectl(args ...string) (string, error) {
	return execKubectl(w.KubeConfigPath(), args...)
}

type waitFn func() error

func (w *Waiter) WaitForAll() error {
	fns := map[string]waitFn{
		"deployments to be Ready": w.WaitForDeployments,
		"DNS to have propagated":  w.WaitForDNSPropagation,
	}
	for desc, fn := range fns {
		msg := fmt.Sprintf("Waiting for %s", desc)
		w.Logger.Infof("%s...", msg)
		before := time.Now().UTC()
		if err := fn(); err != nil {
			return fmt.Errorf("%s failed with: %v", msg, err)
		}
		after := time.Now().UTC()
		w.Logger.Infof("%s took %s", msg, after.Sub(before).String())
	}
	return nil
}

func (w *Waiter) WaitForDeployments() error {
	_, err := w.execKubectl("wait", "-n", "workshopctl", "deployment", "--for=condition=Available", "--all")
	return err
}

func (w *Waiter) WaitForDNSPropagation() error {
	var ip net.IP
	err := util.Poll(nil, w.Logger, func() (bool, error) {
		addr, err := w.execKubectl("-n", "workshopctl", "get", "svc", "traefik", "-otemplate", `--template={{ (index .status.loadBalancer.ingress 0).ip }}`)
		if err != nil {
			return false, err
		}
		ip = net.ParseIP(addr)
		if ip != nil {
			w.Logger.Infof("Got LoadBalancer IP %s for Traefik", ip)
			return true, nil
		}
		return false, fmt.Errorf("no valid IP yet: %q", addr)
	})
	if err != nil {
		return err
	}

	err = util.Poll(nil, w.Logger, func() (bool, error) {
		ips, err := net.LookupIP(w.Domain())
		if err != nil {
			return false, fmt.Errorf("lookup IP error: %v", err)
		}
		// look for the right IP
		for _, addr := range ips {
			if addr.String() == ip.String() {
				return true, nil
			}
		}
		return false, fmt.Errorf("not the right IP found during lookup yet, expected: %s, got: %v", ip, ips)
	})
	if err != nil {
		return err
	}

	// TODO: Make sure Traefik has the LE Cert setup here

	return nil
}
