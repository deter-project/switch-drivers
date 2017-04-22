# Deter Switch Drivers

This repository contains the deter switch drivers. Currently this is only the snmp driver. It may be possible that this is the only driver that is ever required  for conventional switches, as most modern switches (now including Cumulus as of 3.x .... kinda) support the Q-BRIDGE SNMP API for controlling vlans.

The snmp driver is a generic snmp driver that uses the Q-BRIDGE SNMP specification to control vlan configurations on a switch.  At this time it implements the `setPortAccess` and `setPortTrunk` commands as specified in the [deter functional architecture spec](https://github.com/deter-project/spec). It also provides port status query capability and neighbor discovery through LLDP query over SNMP.
