/*~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
 *
 * Deter SNMP Switch Controller Application
 * ========================================
 *
 * This is a command line application that uses the Deter SNMP Switch
 * Controller Library to provide basic switch control. Here is a breif
 * synopsis
 *	usage:
 *		snmp host command
 *		commands:
 *			show
 *			vlan list
 *			vlan create id
 *			vlan delete id
 *			vlan port [index] set access vlan-number
 *			vlan port [index] set trunk [vlan-number]
 *			vlan port [index] clear
 *
 *----------------------------------------------------------
 *
 *			vlan list
 *			interface list
 *
 *			vlan VID set trunk [PORT]
 *			vlan VID set access [PORT]
 *			vlan VID clear [PORT]
 *			vlan VID clear-all
 *
 *			interface INTERFACE set trunk [VID]
 *			interface INTERFACE set access VID
 *			interface INTERFACE clear [VID]
 *			interface INTERFACE clear-all
 *
 *----------------------------------------------------------
 *
 *		examples:
 *			snmp 10.47.1.5 show
 *			snmp 10.47.1.5 vlan create 101
 *			snmp 10.47.1.5 vlan delete 101
 *			snmp 10.47.1.5 vlan port 2 4 6 8 set access 47
 *			snmp 10.47.1.5 vlan port 1 3 5 7 set trunk 101 201 303
 *
 *
 *~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~*/
package main

import (
	"encoding/hex"
	"fmt"
	dsnmp "github.com/deter-project/switch-drivers/snmp/snmp"
	"github.com/fatih/color"
	"log"
	"os"
	"sort"
	"strconv"
)

// Commonly used terminal colors

var blueb = color.New(color.FgBlue, color.Bold).SprintFunc()
var blue = color.New(color.FgBlue).SprintFunc()
var cyan = color.New(color.FgCyan).SprintFunc()
var cyanb = color.New(color.FgCyan, color.Bold).SprintFunc()
var greenb = color.New(color.FgGreen, color.Bold).SprintFunc()
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()
var redb = color.New(color.FgRed, color.Bold).SprintFunc()
var yellow = color.New(color.FgYellow).SprintFunc()
var bold = color.New(color.Bold).SprintFunc()

// *** Entry point ***

func main() {

	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	// get the minimal set of arguments and initialize the switch controller
	args := os.Args[1:]
	if len(args) < 2 {
		log.Fatal(usage())
	}
	host := args[0]
	command := args[1]
	s, err := dsnmp.NewSwitchControllerSnmp(host)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Snmp.Conn.Close()

	// figure out the top level command and execute it
	switch command {
	case "show":
		showSwitch(s)
	case "interface":
		interfaceCmd(s, args[2:])
	case "vlan":
		vlanCmd(s, args[2:])
	default:
		log.Printf("%s %s", red("unknown command"), command)
		log.Fatal(usage())
	}

}

//##
// ### Interface Commands ~~~~~~~
//##
func interfaceCmd(c *dsnmp.SwitchControllerSnmp, args []string) {
	if len(args) == 1 && args[0] == "list" {
		listInterfaces(c)
		return
	}

	bridge_index, err := strconv.Atoi(args[0])
	if err != nil {
		log.Printf("%s %s", red("invalid bridge index"), args[0])
		log.Fatal(usage())
	}

	if len(args) == 2 && args[1] == "clear-all" {
		c.ClearPorts([]int{bridge_index})
		return
	}
	if len(args) >= 3 {
		switch args[1] {
		case "set":
			interfaceSetCmd(c, bridge_index, args[2:])
			return
		case "clear":
			interfaceClearCmd(c, bridge_index, args[2:])
			return
		}
	}
	log.Fatal(usage())
}

func toInts(ss []string) []int {
	vs := make([]int, len(ss))
	for i, a := range ss {
		v, err := strconv.Atoi(a)
		if err != nil {
			log.Printf("%s %s", red("invalid integer value"), a)
			log.Fatal(usage())
		}
		vs[i] = v
	}
	return vs
}

func interfaceSetCmd(c *dsnmp.SwitchControllerSnmp,
	bridge_index int, args []string) {
	if len(args) < 2 {
		log.Fatal(usage())
	}
	vids := toInts(args[1:])
	switch args[0] {
	case "trunk":
		c.SetPortTrunk([]int{bridge_index}, vids)
	case "access":
		c.SetPortAccess([]int{bridge_index}, vids[0])
	}
}

func interfaceClearCmd(c *dsnmp.SwitchControllerSnmp,
	bridge_index int, args []string) {
	vids := make([]int, len(args))
	for i, a := range args {
		vid, err := strconv.Atoi(a)
		if err != nil {
			log.Printf("%s %s", red("invalid vid"), a)
			log.Fatal(usage())
		}
		vids[i] = vid
	}

	c.ClearPortVlans(bridge_index, vids)
}

//##
// ### Vlan Commands ~~~~~~~
//##
func vlanCmd(c *dsnmp.SwitchControllerSnmp, args []string) {

	if len(args) < 1 {
		log.Fatal(usage())
	}

	getNum := func(i int) int {
		if len(args) < i+1 {
			log.Fatal(usage())
		}
		number, err := strconv.Atoi(args[i])
		if err != nil {
			log.Printf("%s %s", red("invalid vlan number"), args[i])
			log.Fatal(usage())
			return -1
		}
		return number
	}

	if len(args) == 1 && args[0] == "list" {
		listVlans(c)
		return
	}

	if len(args) == 2 {
		if args[1] == "clear-all" {
			c.ClearVlans([]int{getNum(0)})
			return
		}
		switch args[0] {
		case "create":
			number := getNum(1)
			err := c.CreateVlan(number)
			if err != nil {
				log.Fatalf("%v", err)
			}
			return
		case "delete":
			number := getNum(1)
			err := c.DeleteVlan(number)
			if err != nil {
				log.Fatalf("%v", err)
			}
			return
		default:
			log.Fatal(usage())
		}
	}

	vid := getNum(0)
	switch args[1] {
	case "set":
		vlanSetCmd(c, vid, args[2:])
	case "clear":
		vlanClearCmd(c, vid, args[2:])
	}

}

func vlanSetCmd(c *dsnmp.SwitchControllerSnmp, vid int, args []string) {

	if len(args) < 2 {
		log.Fatal(usage())
	}
	switch args[0] {
	case "trunk":
		interfaces := toInts(args[1:])
		c.SetPortTrunk(interfaces, []int{vid})
	case "access":
	}

}

func vlanClearCmd(c *dsnmp.SwitchControllerSnmp, vid int, args []string) {
}

// present information to the user on how to use this application
func usage() string {

	meta := fmt.Sprintf("%s %s", blue("snmp"), green("host command"))
	show := fmt.Sprintf("%s", blue("show"))

	vlanList := fmt.Sprintf("%s", blue("vlan list"))
	interfaceList := fmt.Sprintf("%s", blue("interface list"))

	vlanCreate := fmt.Sprintf("%s %s", blue("vlan create"), green("vid"))
	vlanDelete := fmt.Sprintf("%s %s", blue("vlan delete"), green("vid"))

	vlanSetTrunk := fmt.Sprintf("%s %s %s %s",
		blue("vlan"),
		green("vid"),
		blue("set trunk"),
		green("[interface]"))

	vlanSetAccess := fmt.Sprintf("%s %s %s %s",
		blue("vlan"),
		green("vid"),
		blue("set access"),
		green("[interface]"))

	vlanClear := fmt.Sprintf("%s %s %s %s",
		blue("vlan"),
		green("vid"),
		blue("clear"),
		green("[interface]"))

	vlanClearAll := fmt.Sprintf("%s %s %s",
		blue("vlan"),
		green("vid"),
		blue("clear-all"))

	interfaceSetTrunk := fmt.Sprintf("%s %s %s %s",
		blue("interface"),
		green("bridge-index"),
		blue("set trunk"),
		green("[vid]"))

	interfaceSetAccess := fmt.Sprintf("%s %s %s %s",
		blue("interface"),
		green("bridge-index"),
		blue("set access"),
		green("vid"))

	interfaceClear := fmt.Sprintf("%s %s %s %s",
		blue("interface"),
		green("bridge-index"),
		blue("clear"),
		green("[vid]"))

	interfaceClearAll := fmt.Sprintf("%s %s %s",
		blue("interface"),
		green("bridge-index"),
		blue("clear-all"))

	ifFormat := fmt.Sprintf("%s(%s) '%s' %s %s %s",
		bold("[bridge-index]"),
		"device-index",
		"label",
		"type",
		green("admin-status"),
		yellow("op-status"),
	)

	vlanFormat :=
		"vid vlan-name\n" +
			"      egress: [bridge-index list]\n" +
			"      access: [bridge-index list]"

	neighborFormat :=
		"local-device-index <===> remote-host remote-device[mac] remote-uname"

	return redb("\nusage:\n") +
		meta + "\n" +
		"  " + bold("commands:") + " \n" +
		"    " + show + "\n\n" +
		"    " + vlanList + "\n" +
		"    " + vlanCreate + "\n" +
		"    " + vlanDelete + "\n" +
		"    " + vlanSetTrunk + "\n" +
		"    " + vlanSetAccess + "\n" +
		"    " + vlanClear + "\n" +
		"    " + vlanClearAll + "\n\n" +
		"    " + interfaceList + "\n" +
		"    " + interfaceSetTrunk + "\n" +
		"    " + interfaceSetAccess + "\n" +
		"    " + interfaceClear + "\n" +
		"    " + interfaceClearAll + "\n\n" +
		"  " + bold("output format:") + "\n" +
		"    " + blue("interfaces") + "\n" +
		"      " + ifFormat + "\n" +
		"    " + blue("vlans") + "\n" +
		"      " + vlanFormat + "\n" +
		"    " + blue("neighbors") + "\n" +
		"      " + neighborFormat + "\n\n"
}

func maxMe(a *int, b int) {
	if *a < b {
		*a = b
	}
}

type SortedInterfaces []dsnmp.Interface

func (xs SortedInterfaces) Len() int { return len(xs) }
func (xs SortedInterfaces) Less(i, j int) bool {
	return xs[i].BridgeIndex < xs[j].BridgeIndex
}
func (xs SortedInterfaces) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

// produce a textural representation of a switch
func showSwitch(c *dsnmp.SwitchControllerSnmp) {

	ifxs_, err := c.GetInterfaces()
	ifxs := SortedInterfaces(ifxs_)
	sort.Sort(ifxs)

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("\n%s\n", blueb("Interfaces"))
	log.Printf("%s\n", cyanb("=========="))
	for _, v := range ifxs {
		log.Printf(showInterface(v))
	}

	vlans, err := c.GetVlans()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("\n%s\n", blueb("Vlans"))
	log.Printf("%s\n", cyanb("====="))
	for _, v := range vlans {
		log.Printf("%s\n\n", showVlan(v))
	}

	log.Printf("\n%s\n", blueb("Neighbors"))
	log.Printf("%s\n", cyanb("========="))
	nbrs, err := c.GetNeighbors()
	if err != nil {
		log.Fatal(err)
	}

	var widths [2]int
	for _, v := range nbrs {
		maxMe(&widths[0], len(v.RemoteName))
		maxMe(&widths[1], len(v.RemotePortName))
	}

	f :=
		`%2d <==> %-` +
			strconv.Itoa(widths[0]) +
			`s %` +
			strconv.Itoa(widths[1]) +
			`s[%s] '%.64s'`

	for _, v := range nbrs {
		log.Printf(f,
			v.LocalIfIndex,
			v.RemoteName,
			v.RemotePortName,
			hex.EncodeToString(v.RemoteMac),
			v.RemoteDescription,
		)
	}

}

// produce a textual representation of an Interface.
func showInterface(i dsnmp.Interface) string {
	s := fmt.Sprintf("[%d]", i.BridgeIndex)
	if i.BridgeIndex != 0 {
		s = bold(s)
	}
	s += fmt.Sprintf("(%d) '%s' ", i.Index, i.Label)

	if i.Kind == 6 {
		s += "ethernet "
	} else if i.Kind == 161 {
		s += "LAG "
	}

	if i.AdminStatus == 1 {
		s += green("admin ")
	} else if i.AdminStatus == 2 {
		s += red("admin ")
	} else if i.AdminStatus == 3 {
		s += yellow("testing ")
	}

	if i.OpStatus == 1 {
		s += green("op ")
	} else if i.OpStatus == 2 {
		s += red("op ")
	} else if i.OpStatus == 3 {
		s += yellow("op:testing ")
	} else if i.OpStatus == 4 {
		s += yellow("op:unknown ")
	} else if i.OpStatus == 5 {
		s += yellow("op:dormant ")
	} else if i.OpStatus == 6 {
		s += yellow("op:not-present ")
	} else if i.OpStatus == 7 {
		s += yellow("op:lower-down ")
	}

	return s
}

func portmapToString(portmap []byte) string {
	s := ""
	for i := 0; i < len(portmap)*8; i++ {
		if dsnmp.IsPortSet(i, portmap) {
			s += fmt.Sprintf("%d ", i+1)
		}
	}
	return s
}

func portmapMerge(a []byte, b []byte) ([]byte, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("cannot merge portmaps of different lengths")
	}

	c := make([]byte, len(a))

	for i, _ := range a {
		c[i] = a[i] | b[i]
	}

	return c, nil
}

// produce a textual representation of a Vlan.
func showVlan(v dsnmp.Vlan) string {
	s := fmt.Sprintf("%d %s\n", v.Index, v.Name)

	s += "egress ports: "
	for i := 0; i < len(v.EgressPorts)*8; i++ {
		if dsnmp.IsPortSet(i, v.EgressPorts) {
			s += fmt.Sprintf("%d ", i+1)
		}
	}

	s += "\naccess ports: "
	for i := 0; i < len(v.AccessPorts)*8; i++ {
		if dsnmp.IsPortSet(i, v.AccessPorts) {
			s += fmt.Sprintf("%d ", i+1)
		}
	}

	return s
}

func listVlans(c *dsnmp.SwitchControllerSnmp) {
	vlans, err := c.GetVlans()
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range vlans {
		allPorts, err := portmapMerge(v.AccessPorts, v.EgressPorts)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s %d %s", v.Name, v.Index, portmapToString(allPorts))
	}

}

func listInterfaces(c *dsnmp.SwitchControllerSnmp) {
	interfaces, err := c.GetInterfaces()
	if err != nil {
		log.Fatal(err)
	}
	for _, i := range interfaces {
		log.Printf("%d %d %d %d %d",
			i.BridgeIndex,
			i.Index,
			i.Kind,
			i.AdminStatus,
			i.OpStatus,
		)
	}

}

// vlan-port command implementation
/*
func portCmd(c *dsnmp.SwitchControllerSnmp, args []string) {

	for i, p := range args {
		if p == "set" {
			if args[i+1] == "access" {
				accessCmd(c, args[:i], args[i+2])
				return
			} else if args[i+1] == "trunk" {
				trunkCmd(c, args[:i], args[i+2:])
				return
			}
		} else if p == "clear" {
			clearCmd(c, args[:i])
			return
			continue
		}
	}

	log.Fatal(usage())

}
*/

// vlan-port set access command implementation
/*
func accessCmd(c *dsnmp.SwitchControllerSnmp, ports []string, number string) {

	ports_ := extractNumbers(ports)
	number_, err := strconv.Atoi(number)
	if err != nil {
		log.Fatalf("%s: %s is not a valid vlan number", redb("error"))
	}

	err = c.SetPortAccess(ports_, number_)

	if err != nil {
		log.Fatalf("setting port access failed: %v", err)
	}

}
*/

// vlan-port set trunk command implementation
/*
func trunkCmd(c *dsnmp.SwitchControllerSnmp, ports []string, numbers []string) {

	ports_ := extractNumbers(ports)
	numbers_ := extractNumbers(numbers)

	err := c.SetPortTrunk(ports_, numbers_)

	if err != nil {
		log.Fatalf("setting port trunk failed: %v", err)
	}

}
*/

// vlan-port clear command implementation
/*
func clearCmd(c *dsnmp.SwitchControllerSnmp, ports []string) {

	ports_ := extractNumbers(ports)

	err := c.ClearPorts(ports_)

	if err != nil {
		log.Fatalf("clearing port failed: %v", err)
	}

}
*/

// a helper function to turn lists of strings into lists of numbers
/*
func extractNumbers(strings []string) []int {

	ns := []int{}
	for _, p := range strings {
		n, err := strconv.Atoi(p)
		if err != nil {
			fmt.Printf("%s is not a valid port number, skipping\n")
			continue
		}
		ns = append(ns, n)
	}

	return ns
}
*/
