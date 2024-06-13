package main

import (
	"fmt"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/containerd/containerd/services/server/config"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

const (
	hostsConfigTemplate = `[host."https://%s"]
  capabilities = ["pull", "resolve"]`
)

func main() {
	configPath := pflag.String("config-path", "/etc/containerd/config.toml", "containerd config file path")
	registryPluginConfigPath := pflag.String("registry-plugin-config-path", "/etc/containerd/registry.d", "containerd registry plugin config dir path")
	registryPluginHosts := pflag.StringToString("registry-plugin-hosts", map[string]string{"docker.io": "mirror.gcr.io"}, "containerd registry plugin hosts map")
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

	sc, err := dbus.NewSystemdConnection()
	if err != nil {
		log.Fatalf("failed to create systemd connection: %s", err)
	}

	for ticker := time.NewTicker(5 * time.Second); ; {
		select {
		case <-ticker.C:
			log.Debug("check containerd configs consistence")

			cfg := &config.Config{}
			if err := config.LoadConfig(*configPath, cfg); err != nil {
				log.Fatalf("failted to load config: %s", err)
			}
			criPlugin, ok := cfg.Plugins["cri"]
			if !ok {
				log.Fatal("failted to get \"cri\" section")
			}
			_, ok = criPlugin.Get("registry").(*toml.Tree)
			if !ok {
				criPlugin.Set("registry.config_path", *registryPluginConfigPath)
				cfg.Plugins["cri"] = criPlugin

				b, err := toml.Marshal(cfg)
				if err != nil {
					log.Fatalf("failted to serialize \"registry\" section: %s", err)
				}

				if err := os.WriteFile(*configPath, b, 0644); err != nil {
					log.Fatalf("failted to write \"registry\" section: %s", err)
				}

				if _, err := sc.RestartUnit("containerd.service", "replace", nil); err != nil {
					log.Fatalf("failted to restart containerd.service unit: %s", err)
				}

				log.Debug("updated registry config_path value")
			}
			registryConfigPath := criPlugin.Get("registry.config_path").(string)
			for key, value := range *registryPluginHosts {
				dir := path.Join(registryConfigPath, key)
				if err := os.MkdirAll(dir, 0755); err != nil {
					log.Fatalf("failted to create registry hosts config path: %s", err)
				}

				b := []byte(fmt.Sprintf(hostsConfigTemplate, value))
				file := path.Join(dir, "hosts.toml")
				if err := os.WriteFile(file, b, 0644); err != nil {
					log.Fatalf("failted to write registry hosts config file: %s", err)
				}

				log.Debugf("updated or created registry hosts file: %s", file)
			}
		}
	}
}
