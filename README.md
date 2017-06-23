bdmsg-example
======

a [bdmsg](https://github.com/someonegg/bdmsg) example.

Installation
------------

Install bdmsg-example using the "go get" command:

    git clone https://github.com/someonegg/bdmsg-example.git
    
    cd bdmsg-example
    . bin/env.sh
    
    cd test
    go build server/connector
    go build tool/testclient
    
    ./connector
    ./testclient
