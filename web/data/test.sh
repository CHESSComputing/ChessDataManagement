#!/bin/bash
export PATH=$PATH:../../client
srv=http://localhost:8212
docs="test-data.json ID4B-data.json"

for doc in $docs;
do
    schema=`echo $doc | awk '{split($1,a,"-"); print a[1]}'`
    echo "injecting $doc to $srv using schema $schema"
    chess_client -uri=$srv -insert=$doc -schema=$schema -verbose=1
done
