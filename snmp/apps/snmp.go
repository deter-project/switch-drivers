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
	case "vlan":
		vlanCmd(s, args[2:])
	default:
		log.Printf("%s %s", red("unknown command"), command)
		log.Fatal(usage())
	}

}

// present information to the user on how to use this application
func usage() string {

	meta := fmt.Sprintf("%s %s", blue("snmp"), green("host command"))
	show := fmt.Sprintf("%s", blue("show"))
	vlanList := fmt.Sprintf("%s", blue("vlan list"))
	vlanCreate := fmt.Sprintf("%s %s", blue("vlan create"), green("id"))
	vlanDelete := fmt.Sprintf("%s %s", blue("vlan delete"), green("id"))

	setAccess := fmt.Sprintf("%s %s %s %s",
		blue("vlan port"), green("[index]"), blue("set access"), green("vlan-number"))

	setTrunk := fmt.Sprintf("%s %s %s %s",
		blue("vlan port"), green("[index]"), blue("set trunk"), green("[vlan-number]"))

	clearPort := fmt.Sprintf("%s %s %s",
		blue("vlan port"), green("[index]"), blue("clear"))

	return redb("\nusage:\n") +
		meta + "\n" +
		"  " + bold("commands:") + " \n" +
		"    " + show + "\n" +
		"    " + vlanList + "\n" +
		"    " + vlanCreate + "\n" +
		"    " + vlanDelete + "\n" +
		"    " + setAccess + "\n" +
		"    " + setTrunk + "\n" +
		"    " + clearPort + "\n" +
		"  " + bold("examples:") + " \n" +
		"    snmp 10.47.1.5 show\n" +
		"    snmp 10.47.1.5 vlan create 101\n" +
		"    snmp 10.47.1.5 vlan delete 101\n" +
		"    snmp 10.47.1.5 vlan port 2 4 6 8 set access 47\n" +
		"    snmp 10.47.1.5 vlan port 1 3 5 7 set trunk 101 201 303\n\n"
}

func maxMe(a *int, b int) {
	if *a < b {
		*a = b
	}
}

// produce a textural representation of a switch
func showSwitch(c *dsnmp.SwitchControllerSnmp) {

	ifxs, err := c.GetInterfaces()
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
			`s [%s] %s`

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
	s := fmt.Sprintf("%d %s ", i.Index, i.Label)
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
		log.Printf("%d %s", v.Index, v.Name)
	}

}

// top level vlan command implementation
func vlanCmd(c *dsnmp.SwitchControllerSnmp, args []string) {

	if len(args) < 1 {
		log.Fatal(usage())
	}

	getNum := func() int {
		if len(args) < 2 {
			log.Fatal(usage())
		}
		number, err := strconv.Atoi(args[1])
		if err != nil {
			log.Printf("%s %s", red("invalid vlan number"), args[1])
			log.Fatal(usage())
			return -1
		}
		return number
	}

	switch args[0] {
	case "list":
		listVlans(c)
	case "create":
		number := getNum()
		err := c.CreateVlan(number)
		if err != nil {
			log.Fatal("%v", err)
		}
	case "delete":
		number := getNum()
		err := c.DeleteVlan(number)
		if err != nil {
			log.Fatal("%v", err)
		}
	case "port":
		portCmd(c, args[1:])
	default:
		log.Printf("%s %s", red("unknown vlan command"), args[0])
		log.Fatal(usage())
	}

}

// vlan-port command implementation
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

// vlan-port set access command implementation
func accessCmd(c *dsnmp.SwitchControllerSnmp, ports []string, number string) {

	ports_ := extractNumbers(ports)
	number_, err := strconv.Atoi(number)
	if err != nil {
		log.Fatal("%s: %s is not a valid vlan number", redb("error"))
	}

	err = c.SetPortAccess(ports_, number_)

	if err != nil {
		log.Fatal("setting port access failed: %v", err)
	}

}

// vlan-port set trunk command implementation
func trunkCmd(c *dsnmp.SwitchControllerSnmp, ports []string, numbers []string) {

	ports_ := extractNumbers(ports)
	numbers_ := extractNumbers(numbers)

	err := c.SetPortTrunk(ports_, numbers_)

	if err != nil {
		log.Fatal("setting port trunk failed: %v", err)
	}

}

// vlan-port clear command implementation
func clearCmd(c *dsnmp.SwitchControllerSnmp, ports []string) {

	ports_ := extractNumbers(ports)

	err := c.ClearPort(ports_)

	if err != nil {
		log.Fatal("clearing port failed: %v", err)
	}

}

// a helper function to turn lists of strings into lists of numbers
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
