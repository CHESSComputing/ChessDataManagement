### Chess Data Management service

[![Build Status](https://travis-ci.org/vkuznet/ChessDataManagement.svg?branch=master)](https://travis-ci.org/vkuznet/ChessDataManagement)
[![Go Report Card](https://goreportcard.com/badge/github.com/vkuznet/ChessDataManagement)](https://goreportcard.com/report/github.com/vkuznet/ChessDataManagement)
[![GoDoc](https://godoc.org/github.com/vkuznet/ChessDataManagement?status.svg)](https://godoc.org/github.com/vkuznet/ChessDataManagement)

This directory contains codebase for Chess Data Management service.
It is written in Go language and provides functionality described
[here](../README.md):
- kerberos authentication
- handling meta-data in MongoDB

To build it please install Go language on your system
and series of dependencies:

```
# obtain necessary dependencies
go get gopkg.in/mgo.v2
go get github.com/sirupsen/logrus
go get github.com/shirou/gopsutil
go get github.com/divan/expvarmon
go get github.com/sirupsen/logrus
go get github.com/mattn/go-sqlite3
go get github.com/go-sql-driver/mysql
go get -d github.com/shirou/gopsutil/...
go get -d gopkg.in/jcmturner/gokrb5.v7/...
go get gopkg.in/mgo.v2/
go get gopkg.in/mgo.v2/bson

# build server
cd web
go build # or call make
```

To run the service use the following command:
```
web -config server.json
```
where `server.json` has the following form:
```
{
    "uri":"mongodb://localhost:8230",
    "dbname": "chess",
    "dbcoll": "meta",
    "filesdburi": "sqlite3:///path/files.db",
    "port": 8243,
    "templates": "/etc/web/templates",
    "jscripts": "/etc/web/js",
    "styles": "/etc/web/css",
    "images": "/etc/web/images",
    "keytab": "/etc/web/krb5.keytab",
    "krb5Conf": "/etc/web/krb5.conf",
    "realm": "YOUR_KERBEROS_REALM",
    "verbose": 0
}
```

If you prefer, you may run the service via docker:
```
# create /tmp/etc area with your files:
# krb5.keytab, krb5.conf, tls.crt, tls.key, server.json
# run docker container and mount this area to /etc/web
# the default port is 8243
docker run --rm -h `hostname -f` -v /tmp/etc:/etc/web -i -t veknet/chess
```
