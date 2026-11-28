package service

import (
	"context"
	"fmt"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/docker/docker/api/types/container"
	"gopkg.in/yaml.v3"
)

// Settings of an App Store compose service which belong to the user once
// changed. They are written back from the container edit page and kept across
// upgrades. Image, ports, resources and commands stay owned by the app template
// and the app parameter page.
var (
	composeKeyedSettings  = []string{"environment", "labels", "extra_hosts", "volumes"}
	composeScalarSettings = []string{"dns", "hostname", "domainname", "restart", "mac_address", "network_mode"}
)

// preserveComposeCustomizations carries the settings the user changed in the
// installed compose file over to the compose file of the upgrade target.
func preserveComposeCustomizations(install model.AppInstall, next string) (string, error) {
	detail, err := appDetailRepo.GetFirst(repo.WithByID(install.AppDetailId))
	if err != nil {
		return next, err
	}
	if err = ensureAppDetailCompose(install.App, &detail); err != nil {
		return next, err
	}
	base, err := getUpgradeCompose(install, detail)
	if err != nil {
		return next, err
	}
	if base == "" {
		return next, fmt.Errorf("compose template of version %s is not available", install.Version)
	}
	return mergeComposeCustomizations(base, install.DockerCompose, next)
}

// mergeComposeCustomizations is a three-way merge of the user owned settings:
// whatever differs between base and ours is applied to theirs.
func mergeComposeCustomizations(base, ours, theirs string) (string, error) {
	baseMap, oursMap, theirsMap := map[string]interface{}{}, map[string]interface{}{}, map[string]interface{}{}
	for _, item := range []struct {
		content string
		target  *map[string]interface{}
	}{{base, &baseMap}, {ours, &oursMap}, {theirs, &theirsMap}} {
		if err := yaml.Unmarshal([]byte(item.content), item.target); err != nil {
			return theirs, err
		}
	}
	baseServices := composeSection(baseMap, "services")
	oursServices := composeSection(oursMap, "services")
	theirsServices := composeSection(theirsMap, "services")

	changed := false
	for name := range oursServices {
		baseService, inBase := baseServices[name].(map[string]interface{})
		theirsService, inTheirs := theirsServices[name].(map[string]interface{})
		oursService, inOurs := oursServices[name].(map[string]interface{})
		if !inBase || !inTheirs || !inOurs {
			continue
		}
		if mergeComposeService(baseService, oursService, theirsService) {
			changed = true
			declareComposeResources(theirsMap, oursMap, theirsService)
		}
	}
	if !changed {
		return theirs, nil
	}
	content, err := yaml.Marshal(theirsMap)
	if err != nil {
		return theirs, err
	}
	return string(content), nil
}

func mergeComposeService(base, ours, theirs map[string]interface{}) bool {
	changed := false
	for _, key := range composeKeyedSettings {
		baseItems, oursItems := keyedComposeSetting(key, base[key]), keyedComposeSetting(key, ours[key])
		if reflect.DeepEqual(baseItems, oursItems) {
			continue
		}
		theirsItems := keyedComposeSetting(key, theirs[key])
		if mergeKeyed(baseItems, oursItems, theirsItems) {
			setComposeSetting(theirs, key, theirsItems)
			changed = true
		}
	}
	for _, key := range composeScalarSettings {
		baseValue, inBase := base[key]
		oursValue, inOurs := ours[key]
		if inBase == inOurs && reflect.DeepEqual(baseValue, oursValue) {
			continue
		}
		if inOurs {
			theirs[key] = oursValue
		} else {
			delete(theirs, key)
		}
		changed = true
	}

	baseNetworks, oursNetworks := composeServiceNetworks(base["networks"]), composeServiceNetworks(ours["networks"])
	if !reflect.DeepEqual(baseNetworks, oursNetworks) {
		theirsNetworks := composeServiceNetworks(theirs["networks"])
		for name := range unionKeys(baseNetworks, oursNetworks) {
			baseSettings, inBase := baseNetworks[name].(map[string]interface{})
			oursSettings, inOurs := oursNetworks[name].(map[string]interface{})
			theirsSettings, inTheirs := theirsNetworks[name].(map[string]interface{})
			switch {
			case !inOurs:
				if inTheirs {
					delete(theirsNetworks, name)
					changed = true
				}
			case !inBase && !inTheirs:
				theirsNetworks[name] = maps.Clone(oursSettings)
				changed = true
			case inTheirs:
				if mergeKeyed(baseSettings, oursSettings, theirsSettings) {
					changed = true
				}
			case !reflect.DeepEqual(baseSettings, oursSettings):
				theirsNetworks[name] = maps.Clone(oursSettings)
				changed = true
			}
		}
		setComposeNetworks(theirs, theirsNetworks)
	}

	// network_mode and networks are mutually exclusive in compose.
	if _, hasMode := theirs["network_mode"]; hasMode && theirs["networks"] != nil {
		if _, userMode := ours["network_mode"]; userMode {
			delete(theirs, "networks")
		} else {
			delete(theirs, "network_mode")
		}
		changed = true
	}
	return changed
}

// mergeKeyed applies the differences between base and ours to theirs.
func mergeKeyed(base, ours, theirs map[string]interface{}) bool {
	changed := false
	for key := range unionKeys(base, ours) {
		baseValue, inBase := base[key]
		oursValue, inOurs := ours[key]
		if inBase == inOurs && reflect.DeepEqual(baseValue, oursValue) {
			continue
		}
		theirsValue, inTheirs := theirs[key]
		if !inOurs {
			if inTheirs {
				delete(theirs, key)
				changed = true
			}
			continue
		}
		if !inTheirs || !reflect.DeepEqual(theirsValue, oursValue) {
			theirs[key] = oursValue
			changed = true
		}
	}
	return changed
}

func unionKeys(left, right map[string]interface{}) map[string]struct{} {
	keys := make(map[string]struct{}, len(left)+len(right))
	for key := range left {
		keys[key] = struct{}{}
	}
	for key := range right {
		keys[key] = struct{}{}
	}
	return keys
}

// keyedComposeSetting normalizes the list and map syntax of a compose setting.
func keyedComposeSetting(key string, value interface{}) map[string]interface{} {
	items := make(map[string]interface{})
	switch typed := value.(type) {
	case map[string]interface{}:
		for name, item := range typed {
			// List syntax only knows strings, compare both syntaxes alike.
			switch item.(type) {
			case int, int64, uint64, float64, bool:
				item = fmt.Sprint(item)
			}
			items[name] = item
		}
	case []interface{}:
		for _, entry := range typed {
			if key == "volumes" {
				items[composeVolumeTarget(entry)] = entry
				continue
			}
			text := fmt.Sprint(entry)
			separator := strings.Index(text, "=")
			if separator < 0 && key == "extra_hosts" {
				separator = strings.Index(text, ":")
			}
			if separator < 0 {
				items[text] = nil
				continue
			}
			items[text[:separator]] = text[separator+1:]
		}
	}
	return items
}

// setComposeSetting writes a normalized setting back, keeping the syntax and
// order the compose file already uses.
func setComposeSetting(service map[string]interface{}, key string, items map[string]interface{}) {
	if len(items) == 0 {
		delete(service, key)
		return
	}
	original, isList := service[key].([]interface{})
	_, isMap := service[key].(map[string]interface{})
	if isMap || (!isList && (key == "environment" || key == "labels")) {
		service[key] = items
		return
	}
	names := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, entry := range original {
		name := ""
		if key == "volumes" {
			name = composeVolumeTarget(entry)
		} else {
			for item := range keyedComposeSetting(key, []interface{}{entry}) {
				name = item
			}
		}
		if _, ok := items[name]; ok && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	for _, name := range slices.Sorted(maps.Keys(items)) {
		if !seen[name] {
			names = append(names, name)
		}
	}
	separator := "="
	if key == "extra_hosts" {
		separator = ":"
	}
	list := make([]interface{}, 0, len(names))
	for _, name := range names {
		switch {
		case key == "volumes":
			list = append(list, items[name])
		case items[name] == nil:
			list = append(list, name)
		default:
			list = append(list, fmt.Sprintf("%s%s%v", name, separator, items[name]))
		}
	}
	service[key] = list
}

// splitComposeVolume splits the short volume syntax, ignoring colons which are
// part of a variable expression such as ${DIR:-./data}.
func splitComposeVolume(spec string) []string {
	var parts []string
	depth, start := 0, 0
	for index, char := range spec {
		switch char {
		case '{':
			depth++
		case '}':
			depth--
		case ':':
			if depth == 0 {
				parts = append(parts, spec[start:index])
				start = index + 1
			}
		}
	}
	return append(parts, spec[start:])
}

func composeVolumeTarget(entry interface{}) string {
	if long, ok := entry.(map[string]interface{}); ok {
		return fmt.Sprint(long["target"])
	}
	parts := splitComposeVolume(fmt.Sprint(entry))
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[1]
}

func composeNamedVolume(entry interface{}) string {
	source := ""
	if long, ok := entry.(map[string]interface{}); ok {
		if long["type"] == "volume" {
			source, _ = long["source"].(string)
		}
	} else if parts := splitComposeVolume(fmt.Sprint(entry)); len(parts) > 1 {
		source = parts[0]
	}
	if source == "" || strings.ContainsAny(source[:1], "/.~$") {
		return ""
	}
	return source
}

func composeServiceNetworks(value interface{}) map[string]interface{} {
	networks := make(map[string]interface{})
	switch typed := value.(type) {
	case []interface{}:
		for _, name := range typed {
			networks[fmt.Sprint(name)] = map[string]interface{}{}
		}
	case map[string]interface{}:
		for name, raw := range typed {
			settings := map[string]interface{}{}
			if current, ok := raw.(map[string]interface{}); ok {
				maps.Copy(settings, current)
			}
			networks[name] = settings
		}
	}
	return networks
}

func setComposeNetworks(service map[string]interface{}, networks map[string]interface{}) {
	if len(networks) == 0 {
		delete(service, "networks")
		return
	}
	for _, settings := range networks {
		if current, ok := settings.(map[string]interface{}); ok && len(current) > 0 {
			service["networks"] = networks
			return
		}
	}
	list := make([]interface{}, 0, len(networks))
	for _, name := range slices.Sorted(maps.Keys(networks)) {
		list = append(list, name)
	}
	service["networks"] = list
}

func composeSection(composeMap map[string]interface{}, key string) map[string]interface{} {
	section, _ := composeMap[key].(map[string]interface{})
	return section
}

// declareComposeResources adds the top level declarations of networks and named
// volumes the service refers to. Unknown ones already exist on the host.
func declareComposeResources(target, source, service map[string]interface{}) {
	declare := func(section, name string) {
		declared := composeSection(target, section)
		if _, ok := declared[name]; ok {
			return
		}
		if declared == nil {
			declared = map[string]interface{}{}
			target[section] = declared
		}
		if definition, ok := composeSection(source, section)[name]; ok {
			declared[name] = definition
			return
		}
		declared[name] = map[string]interface{}{"external": true}
	}
	for name := range composeServiceNetworks(service["networks"]) {
		if name != "default" {
			declare("networks", name)
		}
	}
	volumes, _ := service["volumes"].([]interface{})
	for _, entry := range volumes {
		if name := composeNamedVolume(entry); name != "" {
			declare("volumes", name)
		}
	}
}

// findAppInstallByContainer returns the App Store installation which owns the
// container, or false when the container is not managed by an installed app.
func findAppInstallByContainer(info container.InspectResponse) (model.AppInstall, bool) {
	if info.Config == nil || info.Config.Labels[composeCreatedBy] != "Apps" {
		return model.AppInstall{}, false
	}
	workdir := info.Config.Labels[composeWorkdirLabel]
	if workdir == "" {
		return model.AppInstall{}, false
	}
	installs, err := appInstallRepo.ListBy(context.Background())
	if err != nil {
		return model.AppInstall{}, false
	}
	for _, install := range installs {
		// vLLM compose files are regenerated by the app itself.
		if install.App.Key != vllmAppKeyForUpgrade && path.Clean(install.GetPath()) == path.Clean(workdir) {
			return install, true
		}
	}
	return model.AppInstall{}, false
}

// syncAppComposeFromContainer persists a container edit into the compose file
// of the installed app, so rebuilds and upgrades keep it.
func syncAppComposeFromContainer(installID uint, labels map[string]string, old, req dto.ContainerOperate) error {
	install, err := appInstallRepo.GetFirst(repo.WithByID(installID))
	if err != nil {
		return err
	}
	content, changed, err := applyContainerEditToCompose(
		install.DockerCompose, labels[composeServiceLabel], labels[composeProjectLabel], old, req,
	)
	if err != nil || !changed {
		return err
	}
	if err = writeUpgradeFile(install.GetComposePath(), []byte(content), constant.FilePerm); err != nil {
		return err
	}
	install.DockerCompose = content
	return appInstallRepo.Save(context.Background(), &install)
}

func applyContainerEditToCompose(content, serviceName, project string, old, req dto.ContainerOperate) (string, bool, error) {
	composeMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(content), &composeMap); err != nil {
		return content, false, err
	}
	service, ok := composeSection(composeMap, "services")[serviceName].(map[string]interface{})
	if !ok {
		return content, false, fmt.Errorf("service %q not found in docker-compose.yml", serviceName)
	}

	// Map docker network names to the keys used by this compose file.
	declared := composeSection(composeMap, "networks")
	networkKeys := map[string]string{project + "_default": "default"}
	for key, raw := range declared {
		definition, _ := raw.(map[string]interface{})
		name, _ := definition["name"].(string)
		if name == "" {
			name = project + "_" + key
			if external, _ := definition["external"].(bool); external {
				name = key
			}
		}
		networkKeys[name] = key
	}
	oldService := containerComposeSettings(old, networkKeys)
	newService := containerComposeSettings(req, networkKeys)
	changed := mergeComposeService(oldService, newService, service)

	// The edit page only shows addresses which were set by hand, so any address
	// in the request is intentional, even when it was set before this edit.
	// A service which lists its networks must also list every attached one.
	listsNetworks := service["networks"] != nil
	for name, raw := range composeServiceNetworks(newService["networks"]) {
		wanted := raw.(map[string]interface{})
		if !listsNetworks && wanted["ipv4_address"] == nil && wanted["ipv6_address"] == nil {
			continue
		}
		networks := composeServiceNetworks(service["networks"])
		settings, exists := networks[name].(map[string]interface{})
		if !exists {
			settings = map[string]interface{}{}
			networks[name] = settings
			changed = true
		}
		for _, key := range []string{"ipv4_address", "ipv6_address"} {
			if value := wanted[key]; value != nil && settings[key] != value {
				settings[key] = value
				changed = true
			}
		}
		setComposeNetworks(service, networks)
		delete(service, "network_mode")
	}

	if !changed {
		return content, false, nil
	}
	declareComposeResources(composeMap, nil, service)
	result, err := yaml.Marshal(composeMap)
	if err != nil {
		return content, false, err
	}
	return string(result), true, nil
}

// containerComposeSettings expresses the user owned fields of the container
// edit form in compose syntax.
func containerComposeSettings(item dto.ContainerOperate, networkKeys map[string]string) map[string]interface{} {
	literal := func(value string) string {
		return strings.ReplaceAll(value, "$", "$$")
	}
	service := map[string]interface{}{}

	pairs := func(values []string) map[string]interface{} {
		result := map[string]interface{}{}
		for _, value := range values {
			key, val, found := strings.Cut(value, "=")
			if key == "" || key == composeCreatedBy || strings.HasPrefix(key, "com.docker.") {
				continue
			}
			if !found {
				val = ""
			}
			result[key] = literal(val)
		}
		return result
	}
	service["environment"] = pairs(item.Env)
	service["labels"] = pairs(item.Labels)

	hosts := map[string]interface{}{}
	for _, host := range item.ExtraHosts {
		if host.Hostname != "" && host.IP != "" {
			hosts[host.Hostname] = literal(host.IP)
		}
	}
	service["extra_hosts"] = hosts

	volumes := []interface{}{}
	for _, volume := range item.Volumes {
		if volume.SourceDir == "" || volume.ContainerDir == "" {
			continue
		}
		mode := volume.Mode
		if mode == "" {
			mode = "rw"
		}
		if volume.Type == "bind" && volume.Shared != "" && volume.Shared != "private" && volume.Shared != "rprivate" {
			mode += "," + volume.Shared
		}
		volumes = append(volumes, literal(volume.SourceDir)+":"+literal(volume.ContainerDir)+":"+mode)
	}
	service["volumes"] = volumes

	var dns []interface{}
	for _, server := range item.DNS {
		if server != "" {
			dns = append(dns, literal(server))
		}
	}
	if len(dns) != 0 {
		service["dns"] = dns
	}
	if item.Hostname != "" {
		service["hostname"] = literal(item.Hostname)
	}
	if item.DomainName != "" {
		service["domainname"] = literal(item.DomainName)
	}
	switch item.RestartPolicy {
	case "", "no":
		service["restart"] = "no"
	case "on-failure":
		service["restart"] = "on-failure:5"
	default:
		service["restart"] = item.RestartPolicy
	}

	networks := map[string]interface{}{}
	for _, endpoint := range item.Networks {
		switch endpoint.Network {
		case "host", "none", "bridge":
			if len(item.Networks) == 1 {
				service["network_mode"] = endpoint.Network
			}
			continue
		}
		name := endpoint.Network
		if key, ok := networkKeys[name]; ok {
			name = key
		}
		settings := map[string]interface{}{}
		if endpoint.Ipv4 != "" {
			settings["ipv4_address"] = endpoint.Ipv4
		}
		if endpoint.Ipv6 != "" {
			settings["ipv6_address"] = endpoint.Ipv6
		}
		if endpoint.MacAddr != "" {
			settings["mac_address"] = endpoint.MacAddr
		}
		networks[name] = settings
	}
	if len(networks) != 0 {
		service["networks"] = networks
	}
	return service
}
