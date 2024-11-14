package hcops

import (
	"strings"
	"strconv"
	"fmt"
	"github.com/hetznercloud/hcloud-go/hcloud"
)

var (
	ProviderName = "hetzner"
	NameLabelType = "node.hetzner.com/type"
	NameCloudNode = "cloud"
	NameDedicatedNode = "dedicated"

	TypeLabels = map[string]map[string]string{
		"cloud": map[string]string{
			"node.hetzner.com/type": "cloud",
			"instance.hetzner.cloud/provided-by": "cloud",
			"instance.hetzner.cloud/is-root-server": "false",
		},
		"dedicated": map[string]string{
			"node.hetzner.com/type": "dedicated",
			"instance.hetzner.cloud/provided-by": "robot",
			"instance.hetzner.cloud/is-root-server": "true",
		},
	}

	ExcludeServer = &hcloud.Server{ 
		ID: 999999,
		ServerType: &hcloud.ServerType{Name: "exclude"},
		Status: hcloud.ServerStatus("running"),
		Datacenter: &hcloud.Datacenter{
			Location: &hcloud.Location{
				Name: "exclude",
			}, 
			Name: "exclude",
		},
	}
)

func ProviderIDToServerID(providerID string) (id int, err error) {
	if providerID == strconv.Itoa(ExcludeServer.ID) {
		return ExcludeServer.ID, nil
	}
	providerPrefix := ProviderName + "://"
	if !strings.HasPrefix(providerID, providerPrefix) {
		err = fmt.Errorf("ERROR: providerID should start with %s://: %s", ProviderName, providerID)
		return
	}

	idString := strings.ReplaceAll(providerID, providerPrefix, "")
	if idString == "" {
		err = fmt.Errorf("ERROR: missing server id in providerID: %s", providerID)
		return
	}

	id, err = strconv.Atoi(idString)
	return
}