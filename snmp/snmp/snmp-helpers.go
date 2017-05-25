/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Deter SNMP Switch Controller Library - Helper Functions
 * ====================================-------------------
 *
 * The code here contains helper functions, mostly low level snmp stuff
 * to make higher level code more readable and maintainable
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
package snmp

import (
	"fmt"
	"github.com/soniah/gosnmp"
	"log"
	"os"
	"strconv"
	"strings"
)

// NewGoSNMP creates a new SNMP Client. Target is the IP address, Community
// the SNMP Community String and Version the SNMP version. Currently only v2c
// is supported. Timeout parameter is measured in seconds.
func NewGoSNMP(
	target, community string,
	version gosnmp.SnmpVersion, timeout int64) (*gosnmp.GoSNMP, error) {

	gosnmp.Default.Target = target
	gosnmp.Default.Community = community
	gosnmp.Default.Version = version
	err := gosnmp.Default.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	return gosnmp.Default, nil
}

// IsPortSet returns whether or not the port at index i is set within the
// object ports which is an snmp style portlist data structure. For the
// details of this structure see RFC 2674 in the Textual Conventions section.
func IsPortSet(i int, ports []byte) bool {

	bits := ports[i/8]
	isSet := bits&(1<<uint(7-(i%8))) > 0
	return isSet

}

// SetPort sets the port at index i in the object ports which is an snmp
// style portlist data structure. For the details of this structure see RFC
// 2674 in the Textual Conventions section.
func SetPort(i int, ports []byte) {

	bits := &ports[i/8]
	bit := 7 - (i % 8)
	*bits |= (1 << uint(bit))

}

// UnsetPort clears the port at index i in the object ports which is an snmp
// style portlist data structure. For the details of this structure see RFC
// 2674 in the Textual Conventions section.
func UnsetPort(i int, ports []byte) {

	bits := &ports[i/8]
	bit := 7 - (i % 8)
	*bits &= ^(1 << uint(bit))

}

// getCounter retrieves a counter object from the device managed by the
// provided snmp object at the provided oid.
func getCounter(snmp *gosnmp.GoSNMP, oid string) (int, error) {

	resp, err := snmp.Get([]string{oid})
	if err == nil {
		for _, v := range resp.Variables {
			switch v.Type {
			case gosnmp.Integer:
				return v.Value.(int), nil
			case gosnmp.Gauge32:
				return int(v.Value.(uint)), nil
			}
		}
	}

	return -1, fmt.Errorf("fail to get counter %s", oid)

}

// walkf retrieves the entire subtree located at the provided oid and
// processes each PDU encounted using the supplied function f
func walkf(
	snmp *gosnmp.GoSNMP,
	oid string,
	kind gosnmp.Asn1BER,
	f func(int, gosnmp.SnmpPDU) error) error {

	resp, err := snmp.BulkWalkAll(oid)
	if err != nil {
		return err
	}
	if err == nil {
		for i, v := range resp {
			switch v.Type {
			case kind:
				err = f(i, v)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// destroyRow marks an snmp table row located at the specified oid
// for destruction.
func destroyRow(snmp *gosnmp.GoSNMP, oid string) error {

	pdu := gosnmp.SnmpPDU{
		Name:   oid,
		Type:   gosnmp.Integer,
		Value:  6,
		Logger: log.New(os.Stdout, "", 0)}

	pkt, err := snmp.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		log.Printf("%#v", pkt)
		return err
	}

	return nil

}

// destroyRow creates a new snmp table row at the provided oid.
func createRow(snmp *gosnmp.GoSNMP, oid string) error {

	pdu := gosnmp.SnmpPDU{
		Name:   oid,
		Type:   gosnmp.Integer,
		Value:  1,
		Logger: log.New(os.Stdout, "", 0)}

	pkt, err := snmp.Set([]gosnmp.SnmpPDU{pdu})

	if err != nil {
		log.Printf("%#v", pkt)
		return err
	}
	if pkt.Error != 0 {
		log.Printf("object creation failed (%d)", pkt.Error)
	}

	return nil

}

// setOctetString sets the value of the octet string located at the
// specified oid
func setOctetString(snmp *gosnmp.GoSNMP, oid string, value []byte) error {

	pdu := gosnmp.SnmpPDU{
		Name:   oid,
		Type:   gosnmp.OctetString,
		Value:  value,
		Logger: log.New(os.Stdout, "", 0)}

	pkt, err := snmp.Set([]gosnmp.SnmpPDU{pdu})

	if err != nil {
		log.Printf("%#v", pkt)
		return err
	}

	return nil

}

/// The following functions provide convinence accessors and legible semantics
/// for frequently used snmp OIDS

const (
	interfaceBridgeIndexOid = ".1.3.6.1.2.1.17.1.4.1.2"
)

func interfacePropertyOid(x int) string {

	return fmt.Sprintf(".1.3.6.1.2.1.2.2.1.%d", x)

}

func currentVlanPropertyOid(x int) string {

	return fmt.Sprintf(".1.3.6.1.2.1.17.7.1.4.2.1.%d", x)

}

func staticVlanPropertyOid(x int) string {

	return fmt.Sprintf(".1.3.6.1.2.1.17.7.1.4.3.1.%d", x)

}

func vlanEgressOid(x int) string {

	return fmt.Sprintf("%s.%d", staticVlanPropertyOid(2), x)

}

func vlanAccessOid(x int) string {

	return fmt.Sprintf("%s.%d", staticVlanPropertyOid(4), x)

}

func extractLLDPIndex(oid string) (int, error) {
	b := strings.LastIndex(oid, ".")
	a := strings.LastIndex(oid[:b], ".")
	i, err := strconv.ParseInt(oid[a+1:b], 10, 0)
	return int(i), err
}
