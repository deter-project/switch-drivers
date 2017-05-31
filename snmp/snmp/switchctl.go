/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Deter SNMP Switch Controller Library - Core Implementation
 * ====================================----------------------
 *
 * This library implements the functionality of the deter switch controller
 * for SNMP based switches. This library requires that the switch implement
 * RFC 2674 (Q-BRIDGE) and support SNMPv2.
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
package snmp

import (
	"fmt"
	"github.com/soniah/gosnmp"
	"math"
	"strconv"
)

///            ----------------------------------------------------------------
/// SwitchControllerSnmp
///  --------------------------------------------------------------------------
type SwitchControllerSnmp struct {
	Snmp *gosnmp.GoSNMP
}

// NewSwitchControllerSNMP creates a new switch controller that controls a
// switch located at the specified address.
func NewSwitchControllerSnmp(address string) (*SwitchControllerSnmp, error) {

	s := new(SwitchControllerSnmp)
	snmp, err := NewGoSNMP(address, "public", gosnmp.Version2c, 5)
	if err != nil {
		return nil, err
	}
	s.Snmp = snmp
	return s, nil

}

// GetInterfaces fetches the interface infrormation from the switch organized
// as a list of Interface objects.
func (c *SwitchControllerSnmp) GetInterfaces() ([]Interface, error) {

	/*
		TODO This fails with snmpd. Have brought this up with the mailing list.
				 Until this gets resolved on that end, using walk instead as per below

		numIfx, err := getCounter(c.Snmp, "1.3.6.1.2.1.2.1.0")
		if err != nil {
			return nil, err
		}
	*/

	//bridge_ifx_map := make(map[int]*Interface)
	//device_ifx_map := make(map[int]*

	numIfx := 0
	err := walkf(
		c.Snmp,
		"1.3.6.1.2.1.2.1.0",
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			numIfx = v.Value.(int)
			return nil
		})

	result := make([]Interface, numIfx)

	if err != nil || numIfx == 0 {
		return result, nil
	}

	devidx := make(map[int]int)

	//indices
	err = walkf(
		c.Snmp,
		interfacePropertyOid(1),
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			idx := v.Value.(int)
			result[i].Index = idx
			devidx[idx] = i
			return nil
		})

	//bridge indices
	err = walkf(
		c.Snmp,
		interfaceBridgeIndexOid,
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			//extract the index from the oid
			idx := v.Value.(int)
			d_idx, ok := devidx[idx]
			if ok {
				suffix := v.Name[len(interfaceBridgeIndexOid)+1:]
				//fmt.Printf("%s - %s\n", v.Name, suffix)
				index, err := strconv.Atoi(suffix)
				if err != nil {
					return err
				}
				result[d_idx].BridgeIndex = index
			}
			return nil
		})

	//labels
	err = walkf(
		c.Snmp,
		interfacePropertyOid(2),
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].Label = string(v.Value.([]byte))
			return nil
		})

	err = walkf(
		c.Snmp,
		".1.3.6.1.2.1.31.1.1.1.18",
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].Label += " " + string(v.Value.([]byte))
			return nil
		})

	//physical layer types
	err = walkf(
		c.Snmp,
		interfacePropertyOid(3),
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].Kind = v.Value.(int)
			return nil
		})

	//admin status
	err = walkf(
		c.Snmp,
		interfacePropertyOid(7),
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].AdminStatus = v.Value.(int)
			return nil
		})

	//operational status
	err = walkf(
		c.Snmp,
		interfacePropertyOid(8),
		gosnmp.Integer,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].OpStatus = v.Value.(int)
			return nil
		})

	return result, nil

}

type Neighbor struct {
	LocalIfIndex      int
	RemoteMac         []byte
	RemoteName        string
	RemotePortName    string
	RemoteDescription string
}

// GetNeighbors fetches the hosts that are directly plugged into the switch.
// At this time we gather this information by reading the LLDP tables.
func (c *SwitchControllerSnmp) GetNeighbors() (map[int]*Neighbor, error) {

	nbrs := make(map[int]*Neighbor)

	//get the macs
	err := walkf(
		c.Snmp,
		".1.0.8802.1.1.2.1.4.1.1.7",
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			mac := v.Value.([]byte)
			i, err := extractLLDPIndex(v.Name)
			if err == nil {
				nbrs[i] = &Neighbor{
					LocalIfIndex: i,
					RemoteMac:    mac,
				}
				return nil
			} else {
				return err
			}
		},
	)
	if err != nil {
		return nbrs, fmt.Errorf("error reading neighbor macs %v", err)
	}

	walkFor := func(oid string, what func(n *Neighbor) *string) {
		walkf(
			c.Snmp,
			oid,
			gosnmp.OctetString,
			func(i int, v gosnmp.SnmpPDU) error {
				value_ := string(v.Value.([]byte))
				i, err := extractLLDPIndex(v.Name)
				if err == nil {
					*(what(nbrs[i])) = value_
					return nil
				} else {
					return err
				}
			},
		)

	}

	//get the hostnames
	walkFor(".1.0.8802.1.1.2.1.4.1.1.9",
		func(n *Neighbor) *string { return &n.RemoteName })

	//get the port names
	walkFor(".1.0.8802.1.1.2.1.4.1.1.8",
		func(n *Neighbor) *string { return &n.RemotePortName })

	//get system description
	walkFor(".1.0.8802.1.1.2.1.4.1.1.10",
		func(n *Neighbor) *string { return &n.RemoteDescription })

	return nbrs, nil
}

// GetVlans fetches the vlan information from the switch organized as a list
// of Vlan objects.
func (c *SwitchControllerSnmp) GetVlans() ([]Vlan, error) {

	numVlan, err := getCounter(c.Snmp, ".1.3.6.1.2.1.17.7.1.1.4.0")
	if err != nil {
		return nil, err
	}

	result := make([]Vlan, numVlan)

	//indices
	err = walkf(
		c.Snmp,
		currentVlanPropertyOid(3),
		gosnmp.Gauge32,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].Index = int(v.Value.(uint))
			return nil
		})

	//egress ports
	err = walkf(
		c.Snmp,
		staticVlanPropertyOid(2),
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].EgressPorts = v.Value.([]byte)
			return nil
		})

	//access ports
	err = walkf(
		c.Snmp,
		staticVlanPropertyOid(4),
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].AccessPorts = v.Value.([]byte)
			return nil
		})

	//names
	err = walkf(
		c.Snmp,
		staticVlanPropertyOid(1),
		gosnmp.OctetString,
		func(i int, v gosnmp.SnmpPDU) error {
			result[i].Name = string(v.Value.([]byte))

			//extract the index from the oid
			suffix := v.Name[len(staticVlanPropertyOid(1))+1:]
			index, err := strconv.Atoi(suffix)
			if err != nil {
				return err
			}
			result[i].Index = index
			return nil
		})

	return result, nil

}

// DeleteVlan removes the specified vlan from the switch under control
func (c *SwitchControllerSnmp) DeleteVlan(number int) error {

	return destroyRow(c.Snmp,
		fmt.Sprintf(".1.3.6.1.2.1.17.7.1.4.3.1.5.%d", number))

}

// CreateVlan creates the specified vlan on the switch under control.
func (c *SwitchControllerSnmp) CreateVlan(number int) error {

	return createRow(c.Snmp,
		fmt.Sprintf(".1.3.6.1.2.1.17.7.1.4.3.1.5.%d", number))

}

// SetPortAccess sets vlan access for the provided vlan number on the
// specified ports.
func (c *SwitchControllerSnmp) SetPortAccess(ports []int, number int) error {

	vlans, err := c.GetVlans()
	if err != nil {
		return fmt.Errorf("SetPortAccess: GetVlans failed: %v", err)
	}

	for _, v := range vlans {
		if v.Index == number {
			for _, p := range ports {
				SetPort(p-1, v.EgressPorts)
				SetPort(p-1, v.AccessPorts)
			}
			setOctetString(c.Snmp, vlanEgressOid(number), v.EgressPorts)
			setOctetString(c.Snmp, vlanAccessOid(number), v.AccessPorts)
			return nil
		}
	}

	//an existing vlan as not found, so let's make one
	bridge_size, err := getCounter(c.Snmp, ".1.3.6.1.2.1.17.1.2.0")
	if err != nil {
		return err
	}
	portmap_size := int(math.Ceil(float64(bridge_size) / 8.0))
	portmap := make([]byte, portmap_size)
	for _, p := range ports {
		SetPort(p-1, portmap)
		SetPort(p-1, portmap)
	}
	setOctetString(c.Snmp, vlanEgressOid(number), portmap)
	setOctetString(c.Snmp, vlanAccessOid(number), portmap)
	return nil

	//return fmt.Errorf("vlan %d does not exist", number)

}

// SetPortAccess sets a vlan trunk for the provided vlan numbers on the
// specified ports on the switch under control.
func (c *SwitchControllerSnmp) SetPortTrunk(ports []int, numbers []int) error {

	vlans, err := c.GetVlans()
	if err != nil {
		return fmt.Errorf("SetPortTrunk: GetVlans failed: %v", err)
	}

	for _, number := range numbers {
		for _, v := range vlans {
			if v.Index == number {
				for _, p := range ports {
					SetPort(p-1, v.EgressPorts)
				}
				setOctetString(c.Snmp, vlanEgressOid(number), v.EgressPorts)
				setOctetString(c.Snmp, vlanAccessOid(number), v.AccessPorts)
			}
		}
	}

	return nil

}

// ClearPort clears the specified ports of any an all vlans on the switch
// under control.
func (c *SwitchControllerSnmp) ClearPort(ports []int) error {

	vlans, err := c.GetVlans()
	if err != nil {
		return fmt.Errorf("ClearPort: GetVlans failed: %v", err)
	}

	for _, v := range vlans {
		for _, p := range ports {
			UnsetPort(p-1, v.EgressPorts)
			UnsetPort(p-1, v.AccessPorts)
		}
		setOctetString(c.Snmp, vlanEgressOid(v.Index), v.EgressPorts)
		setOctetString(c.Snmp, vlanAccessOid(v.Index), v.AccessPorts)
	}

	return nil

}

// An Interface represents an interface on a switch.
type Interface struct {
	Label                                           string
	Index, BridgeIndex, Kind, AdminStatus, OpStatus int
}

// A Vlan represents an 802.1Q virtual lan bridge object on a switch
type Vlan struct {
	Index                    int
	EgressPorts, AccessPorts []byte
	Name                     string
}
