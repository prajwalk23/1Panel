package service

import (
	"strings"
	"testing"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
)

const customComposeBase = `services:
  mail:
    image: mail:1.0
    environment:
      - TZ=UTC
      - SIZE=10
    networks:
      - 1panel-network
    ports:
      - ${HOST_IP}:8081:80
    restart: always
    volumes:
      - ./data:/var/www
networks:
  1panel-network:
    external: true
`

func customComposeForm() dto.ContainerOperate {
	return dto.ContainerOperate{
		Env:           []string{"TZ=UTC", "SIZE=10", "PATH=/usr/bin"},
		Labels:        []string{"createdBy=Apps", "com.docker.compose.service=mail"},
		Hostname:      "a884f6a26689",
		RestartPolicy: "always",
		Networks:      []dto.ContainerNetwork{{Network: "1panel-network", MacAddr: "1e:45:cf:6f:e4:fb"}},
		Volumes:       []dto.VolumeHelper{{Type: "bind", SourceDir: "/opt/app/data", ContainerDir: "/var/www", Mode: "rw"}},
	}
}

func applyCustomEdit(t *testing.T, old, req dto.ContainerOperate) (string, bool) {
	t.Helper()
	content, changed, err := applyContainerEditToCompose(customComposeBase, "mail", "mail", old, req)
	if err != nil {
		t.Fatal(err)
	}
	return content, changed
}

func assertContains(t *testing.T, content string, wanted []string, unwanted []string) {
	t.Helper()
	for _, item := range wanted {
		if !strings.Contains(content, item) {
			t.Errorf("missing %q in:\n%s", item, content)
		}
	}
	for _, item := range unwanted {
		if strings.Contains(content, item) {
			t.Errorf("unexpected %q in:\n%s", item, content)
		}
	}
}

func TestContainerEditWithoutChangesKeepsCompose(t *testing.T) {
	content, changed := applyCustomEdit(t, customComposeForm(), customComposeForm())
	if changed || content != customComposeBase {
		t.Fatalf("compose changed:\n%s", content)
	}
}

func TestContainerEditSavesStaticIP(t *testing.T) {
	req := customComposeForm()
	req.Networks[0].Ipv4 = "172.18.0.11"
	content, changed := applyCustomEdit(t, customComposeForm(), req)
	if !changed {
		t.Fatal("static IP was not saved")
	}
	assertContains(t, content,
		[]string{"ipv4_address: 172.18.0.11", "external: true"},
		[]string{"mac_address", "PATH=", "hostname", "/opt/app/data"})

	// An address set before this edit is saved too.
	if _, changed = applyCustomEdit(t, req, req); !changed {
		t.Fatal("existing static IP was not saved")
	}
}

func TestContainerEditSavesChangedSettings(t *testing.T) {
	req := customComposeForm()
	req.Env = []string{"TZ=UTC", "SIZE=20", "PATH=/usr/bin", "SECRET=a$b"}
	req.Volumes = append(req.Volumes, dto.VolumeHelper{Type: "volume", SourceDir: "mail-extra", ContainerDir: "/extra", Mode: "ro"})
	req.ExtraHosts = []dto.ExtraHost{{Hostname: "db", IP: "10.0.0.2"}}
	req.DNS = []string{"1.1.1.1"}
	req.Networks = append(req.Networks, dto.ContainerNetwork{Network: "backend"})
	content, _ := applyCustomEdit(t, customComposeForm(), req)
	assertContains(t, content,
		[]string{"SIZE=20", "SECRET=a$$b", "./data:/var/www", "mail-extra:/extra:ro", "db:10.0.0.2", "1.1.1.1", "- backend", "- 1panel-network", "mail-extra:\n"},
		[]string{"PATH=", "SIZE=10"})
}

func TestMergeComposeCustomizations(t *testing.T) {
	req := customComposeForm()
	req.Networks[0].Ipv4 = "172.18.0.11"
	req.Env = append(req.Env, "ADDED=1")
	ours, _ := applyCustomEdit(t, customComposeForm(), req)
	next := strings.NewReplacer("mail:1.0", "mail:2.0", "SIZE=10", "SIZE=50", "./data:/var/www", "./data:/srv/www").Replace(customComposeBase)

	merged, err := mergeComposeCustomizations(customComposeBase, ours, next)
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, merged,
		[]string{"mail:2.0", "SIZE=50", "ADDED=1", "ipv4_address: 172.18.0.11", "./data:/srv/www"},
		[]string{"mail:1.0", "SIZE=10", "./data:/var/www"})

	unchanged, err := mergeComposeCustomizations(customComposeBase, customComposeBase, next)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged != next {
		t.Fatalf("untouched install changed the new compose:\n%s", unchanged)
	}
}
