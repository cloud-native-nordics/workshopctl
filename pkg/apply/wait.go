package apply

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/cloud-native-nordics/workshopctl/pkg/config"
	"github.com/cloud-native-nordics/workshopctl/pkg/constants"
	"github.com/cloud-native-nordics/workshopctl/pkg/util"
	"github.com/sirupsen/logrus"
)

type Waiter struct {
	*config.ClusterInfo
	ctx    context.Context
	logger *logrus.Entry
	dryrun bool
}

func NewWaiter(ctx context.Context, info *config.ClusterInfo, dryrun bool) *Waiter {
	// TODO: Make dry-run a context-scoped variable
	return &Waiter{info, ctx, util.Logger(ctx), dryrun}
}

func (w *Waiter) kubectl() *kubectlExecer {
	return kubectl(w.KubeConfigPath(), w.dryrun).WithNS(constants.WorkshopctlNamespace)
}

type waitFn func() error

func (w *Waiter) WaitForAll() error {
	fns := map[string]waitFn{
		"deployments to be Ready": w.WaitForDeployments,
		"DNS to have propagated":  w.WaitForDNSPropagation,
		//"TLS certs to have been created": w.WaitForTLSSetup,
	}
	for desc, fn := range fns {
		msg := fmt.Sprintf("Waiting for %s", desc)
		w.logger.Infof("%s...", msg)
		before := time.Now().UTC()
		if err := fn(); err != nil {
			return fmt.Errorf("%s failed with: %v", msg, err)
		}
		after := time.Now().UTC()
		w.logger.Infof("%s took %s", msg, after.Sub(before).String())
	}
	return nil
}

func (w *Waiter) WaitForDeployments() error {
	return util.Poll(nil, w.logger, func() (bool, error) {
		// Wait 30s using kubectl until the "global" Poll timeout is reached
		_, err := w.kubectl().WithArgs("wait", "deployment", "--for=condition=Available", "--all", "--timeout=30s").Run()
		if err != nil {
			return false, err
		}
		return true, nil
	}, w.dryrun)
}

func (w *Waiter) WaitForDNSPropagation() error {
	var ip net.IP
	err := util.Poll(nil, w.logger, func() (bool, error) {
		addr, err := w.kubectl().WithArgs("get", "svc", "traefik", "-otemplate", `--template={{ (index .status.loadBalancer.ingress 0).ip }}`).Run()
		if err != nil {
			return false, err
		}
		ip = net.ParseIP(addr)
		if ip != nil {
			w.logger.Infof("Got LoadBalancer IP %s for Traefik", ip)
			return true, nil
		}
		return false, fmt.Errorf("no valid IP yet: %q", addr)
	}, w.dryrun)
	if err != nil {
		return err
	}

	return util.Poll(nil, w.logger, func() (bool, error) {
		prefixes := []string{""} // "dashboard"
		for _, prefix := range prefixes {
			domain := w.Domain()
			if len(prefix) > 0 {
				domain = fmt.Sprintf("%s.%s", prefix, w.Domain())
			}
			if err := domainMatches(domain, ip); err != nil {
				return false, err
			}
			w.logger.Infof("%s now resolves to %s, as expected", domain, ip)
		}

		return true, nil
	}, w.dryrun)
}

func domainMatches(domain string, expectedIP net.IP) error {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}
	ips, err := r.LookupIPAddr(context.Background(), domain)
	if err != nil {
		return fmt.Errorf("Domain lookup error for %q: %v", domain, err)
	}
	// look for the right IP
	for _, addr := range ips {
		if addr.IP.String() == expectedIP.String() {
			return nil
		}
	}
	return fmt.Errorf("Not the right IP found during lookup yet, expected: %s, got: %v", expectedIP, ips)
}

func (w *Waiter) WaitForTLSSetup() error {
	// TODO: Somehow verify if Traefik already has got the TLS cert
	_, err := w.kubectl().WithArgs("delete", "pod", "-l=app=traefik").Run()
	if err != nil {
		return err
	}
	w.logger.Infof("Restarted traefik")
	return nil
	/*
		TODO: Maybe verify somehow that we can connect to the endpoint(s) correctly.
		return util.Poll(nil, w.logger, func() (bool, error) {
			_, err := http.Get(w.Domain())
			return (err == nil), err
		}, w.dryrun)
	*/
}
