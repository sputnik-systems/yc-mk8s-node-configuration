// just for testing inaccessible docker.io
package main

import (
	"fmt"
	"net"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

func main() {
	logLevel := pflag.String("log-level", "info", "daemon log level")
	chrootPath := pflag.String("chroot-path", "/host", "path, where host filesystem mounted")
	pflag.Parse()

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("failed to parse log level value override: %s", err)
	}
	log.SetLevel(level)

	if err := syscall.Chroot(*chrootPath); err != nil {
		log.Fatalf("failed to chroot in host filesystem: %s", err)
	}

	c, err := iptables.New()
	if err != nil {
		panic(err)
	}
	addrs, err := net.LookupIP("registry-1.docker.io")
	if err != nil {
		panic(err)
	}
	rules := make([][]string, 0)
	for _, addr := range addrs {
		if addr.To4() == nil {
			continue
		}

		rule := []string{"-d", fmt.Sprintf("%s/32", addr.String()), "-j", "DROP"}
		rules = append(rules, rule)
	}

	for _, rule := range rules {
		log.Debugf("adding rule into iptables ruleset: %+v", rule)

		if err := c.InsertUnique("filter", "OUTPUT", 1, rule...); err != nil {
			panic(err)
		}
	}
}
