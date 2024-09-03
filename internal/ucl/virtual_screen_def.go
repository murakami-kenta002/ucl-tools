package ucl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

const VSCRNDEF_FILE = "/etc/uhmi-framework/virtual-screen-def.json"

type VScrnDef struct {
	Nodes []struct {
		NodeId   int    `json:"node_id"`
		HostName string `json:"hostname"`
		Ip       string `json:"ip"`
	} `json:"node"`

	DistrubutedWindowSystem struct {
		FrameworkNode []struct {
			NodeId int `json:"node_id"`
			Ucl    struct {
				Port int `json:"port"`
			} `json:"ucl"`
		} `json:"framework_node"`
	} `json:"distributed_window_system"`
}

type UclNode struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

func ReadVScrnDef(fname string) (*VScrnDef, error) {

	f, err := os.Open(fname)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("%s %s", fname, err))
	}

	jsonBytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ReadAll error: ", err))
	}

	var vscrnDef VScrnDef
	err = json.Unmarshal(jsonBytes, &vscrnDef)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("json Unmarshal error: %s", err))
	}

	return &vscrnDef, nil
}

func GetIpAddr(nodeId int, vscrnDef *VScrnDef) (string, error) {

	for _, r := range vscrnDef.Nodes {
		if nodeId == r.NodeId {
			return r.Ip, nil
		}
	}

	return "0.0.0.0", errors.New("Cannot Find My IP Address from VScrnDef json")
}

func GetNodeIdAndIpAddr(ipAddrs []string, hostname string, vscrnDef *VScrnDef) (int, string, error) {

	for _, r := range vscrnDef.Nodes {
		if hostname == r.HostName {
			for _, ipaddr := range ipAddrs {
				if ipaddr == r.Ip {
					return r.NodeId, r.Ip, nil
				}
			}
		}
	}

	return -1, "0.0.0.0", errors.New("Cannot Find My NodeId and IP Address from VScrnDef json")
}

func GetPort(nodeId int, vscrnDef *VScrnDef) (int, error) {

	for _, r := range vscrnDef.DistrubutedWindowSystem.FrameworkNode {
		if nodeId == r.NodeId {
			return r.Ucl.Port, nil
		}
	}

	return -1, errors.New("Cannot Find My Port from VScrnDef json")
}
