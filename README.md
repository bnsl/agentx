# agentx
A implementation of the Agent X protocol from RFC2741 in the Go language

This package implements a (current subset) of RFC2741 - the agentx protocol.

At the moment it implements a client with support for the following:
* Open and Register Tree
* Get SNMP Object
* GetNext SNMP objects

Other functionality is not provided at this point.

It also includes a lot of (free) debugging output.

TODO
----

* Write docs and work out what should and shouldn't be exported.
* Complete implementation of Client.
* Implement Server (ie, agentX master)
* Implement all types.
