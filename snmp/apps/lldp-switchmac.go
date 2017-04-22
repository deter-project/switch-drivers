/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Deter lldp-switchmac application
 * ================================
 *
 *	This application is purpose built to provide the web interface with a
 *	list of mac addresses connected to a switch in a format it expects. This
 *  format is
 *
 *  <mac>,<switch>/<module>.<port>,<vlan>,<interface>,<class>
 *
 *	usage:
 *		switchmac <switch-address> [experimental|control]
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
package main

import (
	"encoding/hex"
	"fmt"
	dsnmp "github.com/deter-project/switch-drivers/snmp/snmp"
	"log"
	"os"
)

const CONTROL_VLAN int = 2003

func main() {

	//no timestamp on logging
	log.SetFlags(0)

	//grab the hostname from args
	if len(os.Args) < 3 {
		log.Fatal(usage())
	}
	host := os.Args[1]
	class := os.Args[2]

	//create a new instance of the switch controller
	s, err := dsnmp.NewSwitchControllerSnmp(host)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Snmp.Conn.Close()

	//ask the switch who it's neighbors are
	nbrs, err := s.GetNeighbors()
	if err != nil {
		log.Fatal(err)
	}

	//plop out the expected format
	for _, n := range nbrs {
		fmt.Printf("%s,%s/%d.%d,%d,%s:%s,%s\n",
			hex.EncodeToString(n.RemoteMac),
			host,
			0,
			n.LocalIfIndex,
			CONTROL_VLAN,
			n.RemoteName, n.RemotePortName,
			class,
		)
	}

}

func usage() string {
	return "usage:\n  switchmac <switch-address> [experimental|control]"
}
