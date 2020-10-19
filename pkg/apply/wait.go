package apply

import (
	"fmt"
	"net"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
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
		"deployments to be Ready":        w.WaitForDeployments,
		"DNS to have propagated":         w.WaitForDNSPropagation,
		"TLS certs to have been created": w.WaitForTLSSetup,
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
	return util.Poll(nil, w.Logger, func() (bool, error) {
		// Wait 30s using kubectl until the "global" Poll timeout is reached
		_, err := w.execKubectl("wait", "-n", "workshopctl", "deployment", "--for=condition=Available", "--all", "--timeout=30s")
		if err != nil {
			return false, err
		}
		return true, nil
	})
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

	return util.Poll(nil, w.Logger, func() (bool, error) {
		prefixes := []string{"", "dashboard"}
		for _, prefix := range prefixes {
			domain := w.Domain()
			if len(prefix) > 0 {
				domain = fmt.Sprintf("%s.%s", prefix, w.Domain())
			}
			if err := domainMatches(domain, ip); err != nil {
				return false, err
			}
			w.Logger.Infof("%s now resolves to %s, as expected", domain, ip)
		}

		return true, nil
	})
}

func domainMatches(domain string, expectedIP net.IP) error {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return fmt.Errorf("Domain lookup error for %q: %v", domain, err)
	}
	// look for the right IP
	for _, addr := range ips {
		if addr.String() == expectedIP.String() {
			return nil
		}
	}
	return fmt.Errorf("Not the right IP found during lookup yet, expected: %s, got: %v", expectedIP, ips)
}

func (w *Waiter) WaitForTLSSetup() error {
	// TODO: Somehow verify if Traefik already has got the TLS cert
	_, err := w.execKubectl("-n", "workshopctl", "delete", "pod", "-l=app=traefik")
	if err != nil {
		return err
	}
	w.Logger.Infof("Restarted traefik")
	return nil
	/*
		TODO: Maybe verify somehow that we can connect to the endpoint(s) correctly.
		return util.Poll(nil, w.Logger, func() (bool, error) {
			_, err := http.Get(w.Domain())
			return (err == nil), err
		})
	*/
}
