package hcops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var (
	ProviderName      = "hetzner"
	NameLabelType     = "node.hetzner.com/type"
	NameCloudNode     = "cloud"
	NameDedicatedNode = "dedicated"

	TypeLabels = map[string]map[string]string{
		"cloud": {
			"node.hetzner.com/type":               "cloud",
			"instance.hetzner.cloud/provided-by":  "cloud",
			"instance.hetzner.cloud/is-root-server": "false",
		},
		"dedicated": {
			"node.hetzner.com/type":                 "dedicated",
			"instance.hetzner.cloud/provided-by":    "robot",
			"instance.hetzner.cloud/is-root-server": "true",
		},
	}

	ExcludeServer = &hcloud.Server{
		ID:         999999,
		ServerType: &hcloud.ServerType{Name: "exclude"},
		Status:     hcloud.ServerStatus("running"),
		Location: &hcloud.Location{
			Name: "exclude",
		},
	}
)

// RobotDatacenterLabel stores the original Robot DC name on synthetic servers
// so zone labels stay stable after Server.Datacenter was removed from the API.
const RobotDatacenterLabel = "ccm.hetzner.local/robot-datacenter"

func ProviderIDToServerID(providerID string) (id int64, err error) {
	if providerID == strconv.FormatInt(ExcludeServer.ID, 10) {
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

	id, err = strconv.ParseInt(idString, 10, 64)
	return
}
