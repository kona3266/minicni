package handler

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/morvencao/minicni/pkg/args"
	"github.com/morvencao/minicni/pkg/nettool"
	"github.com/morvencao/minicni/pkg/version"

	"github.com/containernetworking/plugins/pkg/ns"
)

type FileHandler struct {
	*version.VersionInfo
}

func NewFileHandler() Handler {
	return &FileHandler{
		VersionInfo: &version.VersionInfo{
			CniVersion:        version.Version,
			SupportedVersions: []string{version.Version},
		},
	}
}

func (fh *FileHandler) HandleAdd(cmdArgs *args.CmdArgs) error {
	cniConfig := args.CNIConfiguration{}
	if err := json.Unmarshal(cmdArgs.StdinData, &cniConfig); err != nil {
		return err
	}

	gwIP, gw_addr, _ := nettool.GetFirstIp(cniConfig.Subnet)
	if err := nettool.IpAmfs.SetIpUsed(gw_addr); err != nil {
		return err
	}

	podIP, err := nettool.IpAmfs.AllocIp(cniConfig.Subnet)
	if err != nil {
		return err
	}

	// Create or update bridge
	brName := cniConfig.Bridge
	if brName != "" {
		// fall back to default bridge name: minicni0
		brName = "minicni0"
	}
	mtu := cniConfig.MTU
	if mtu == 0 {
		// fall back to default MTU: 1500
		mtu = 1500
	}
	br, err := nettool.CreateOrUpdateBridge(brName, gwIP, mtu)
	if err != nil {
		return err
	}

	netns, err := ns.GetNS(cmdArgs.Netns)
	if err != nil {
		return err
	}

	if err := nettool.SetupVeth(netns, br, cmdArgs.IfName, podIP, gwIP, mtu); err != nil {
		return err
	}

	addCmdResult := &AddCmdResult{
		CniVersion: cniConfig.CniVersion,
		Interfaces: []nettool.Interface{{Name: cmdArgs.IfName}},
		IPs: []nettool.AllocatedIP{
			{Version: "4", Address: podIP, Gateway: gw_addr, Interface: 0},
		}}

	addCmdResultBytes, err := json.Marshal(addCmdResult)
	if err != nil {
		return err
	}

	// kubelet expects json format from stdout if success
	fmt.Print(string(addCmdResultBytes))

	return nil
}

func (fh *FileHandler) HandleDel(cmdArgs *args.CmdArgs) error {
	netns, err := ns.GetNS(cmdArgs.Netns)
	cniConfig := args.CNIConfiguration{}
	if err := json.Unmarshal(cmdArgs.StdinData, &cniConfig); err != nil {
		return err
	}

	if err != nil {
		return err
	}
	ip, err := nettool.GetVethIPInNS(netns, cmdArgs.IfName)
	if err != nil {
		return err
	}
	ipn, _, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	nettool.IpAmfs.ReleaseIp(cniConfig.Subnet, ipn)

	return nil
}

func (fh *FileHandler) HandleCheck(cmdArgs *args.CmdArgs) error {
	// to br implemented
	return nil
}

func (fh *FileHandler) HandleVersion(cmdArgs *args.CmdArgs) error {
	versionInfo, err := json.Marshal(fh.VersionInfo)
	if err != nil {
		return err
	}
	fmt.Print(string(versionInfo))
	return nil
}
