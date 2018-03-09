all: \
	build/lldp-switchmac \
	build/snmpd

build/lldp-switchmac: snmp/apps/lldp-switchmac.go snmp/snmp/*.go | build
	go build -o $@ $<

build/snmpd: snmp/apps/snmp.go snmp/snmp/*.go | build
	go build -o $@ $<

build:
	mkdir build

clean:
	rm -rf build

