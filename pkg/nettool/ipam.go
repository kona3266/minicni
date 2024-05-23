package nettool

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

type ipAmFs struct {
	subnets map[string]*bitMap
	path    string
}

const (
	IpAmStorageFsPath = "/tmp/reserved_ips"
)

var IpAmfs = &ipAmFs{
	subnets: make(map[string]*bitMap),
	path:    IpAmStorageFsPath,
}

func init() {
	IpAmfs.InitNetwork()
}

func (ipamfs *ipAmFs) InitNetwork() error {
	if _, err := os.Stat(ipamfs.path); err != nil {
		if os.IsNotExist(err) {
			os.Create(ipamfs.path)
		} else {
			return err
		}
	}

	return nil
}

func (ipamfs *ipAmFs) loadConf() error {
	content, err := os.ReadFile(ipamfs.path)
	if err != nil {
		return err
	}

	if len(content) != 0 {
		err = json.Unmarshal(content, &ipamfs.subnets)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ipamfs *ipAmFs) ReleaseIp(subnet string, ip net.IP) error {
	if err := ipamfs.loadConf(); err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}
	bitmap := ipamfs.subnets[cidr.String()]
	if bitmap == nil {
		return nil
	}
	pos := getIPIndex(ip, cidr.Mask)
	bitmap.BitClean(pos)
	return ipamfs.sync()
}

func (ipamfs *ipAmFs) AllocIp(subnet string) (string, error) {
	if err := ipamfs.loadConf(); err != nil {
		return "", err
	}
	ip, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		return "", err
	}
	ip = ip.To4()
	// 24, 32 => "192.168.0.0/24"
	ones, total := cidr.Mask.Size()
	bitmap := ipamfs.subnets[cidr.String()]
	if bitmap == nil || bitmap.Bitmap == nil {
		bitmap = InitBitMap(1 << (total - ones))
		ipamfs.subnets[cidr.String()] = bitmap
	}

	allocated := false

	// 如果掩码是24，pos 为0 是网络号不能分配ip，pos 255是广播地址。因此遍历1-254
	for pos := 1; pos <= (1<<(total-ones) - 2); pos++ {
		if bitmap.BitExist(pos) {
			continue
		}
		allocated = true
		bitmap.BitSet(pos)
		firstIP := ipToUint32(ip.Mask(cidr.Mask))
		ip = uint32ToIP(firstIP + uint32(pos))
		break
	}

	if !allocated {
		return "", fmt.Errorf("no IP available")
	}

	err = ipamfs.sync()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

func (ipamfs *ipAmFs) sync() error {
	if _, err := os.Stat(ipamfs.path); err != nil {
		if os.IsNotExist(err) {
			os.Create(ipamfs.path)
		} else {
			return err
		}
	}
	data, err := json.Marshal(ipamfs.subnets)
	if err != nil {
		return err
	}
	err = os.WriteFile(ipamfs.path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func ipToUint32(ip net.IP) uint32 {
	if ip == nil {
		return 0
	}
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return binary.BigEndian.Uint32(ip)
}

func uint32ToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

func (ipamfs *ipAmFs) SetIpUsed(cidr_string string) error {
	if err := ipamfs.loadConf(); err != nil {
		return err
	}
	ip, cidr, err := net.ParseCIDR(cidr_string)
	if err != nil {
		return err
	}
	ip = ip.To4()
	ones, total := cidr.Mask.Size()
	bitmap := ipamfs.subnets[cidr.String()]
	if bitmap == nil || bitmap.Bitmap == nil {
		bitmap = InitBitMap(1 << (total - ones))
		ipamfs.subnets[cidr.String()] = bitmap
	}
	pos := getIPIndex(ip, cidr.Mask)
	log.Printf("set  ip %s pos %d \n", ip, pos)
	bitmap.BitSet(pos)
	return ipamfs.sync()
}

func getIPIndex(ip net.IP, mask net.IPMask) int {
	ipInt := ipToUint32(ip)
	firstIP := ipToUint32(ip.Mask(mask))
	return int(ipInt - firstIP)
}
