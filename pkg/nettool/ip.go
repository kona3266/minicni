package nettool

import (
	"net"
	"strconv"
)

type AllocatedIP struct {
	Version   string `json:"version"`
	Address   string `json:"address"`
	Gateway   string `json:"gateway"`
	Interface int64  `json:"interface"`
}

type Interface struct {
	Name string `json:"name"`
}

func GetFirstIp(cidr string) (string, string, error) {
	ip, net, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", err
	}

	maskLen, _ := net.Mask.Size()
	ip = ip.To4()
	ip[len(ip)-1]++
	return ip.String(), ip.String() + "/" + strconv.Itoa(maskLen), nil
}
