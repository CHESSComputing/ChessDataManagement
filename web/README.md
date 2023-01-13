### Chess Data Management service

[![Build Status](https://travis-ci.org/vkuznet/ChessDataManagement.svg?branch=master)](https://travis-ci.org/vkuznet/ChessDataManagement)
[![Go Report Card](https://goreportcard.com/badge/github.com/vkuznet/ChessDataManagement)](https://goreportcard.com/report/github.com/vkuznet/ChessDataManagement)
[![GoDoc](https://godoc.org/github.com/vkuznet/ChessDataManagement?status.svg)](https://godoc.org/github.com/vkuznet/ChessDataManagement)

This directory contains codebase for Chess Data Management service.
It is written in Go language and provides functionality described
[here](../README.md):
- kerberos authentication
- handling meta-data in MongoDB

To build server code please use make command:
```
make
```

To run the service use the following command:
```
web -config server.json
```
For server configuration parameters please refer to
[server.json](server_test.json) and/or [Configuration](config.go)
data-structure.

If you prefer, you may run the service via docker:
```
# create /tmp/etc area with your files:
# krb5.keytab, krb5.conf, tls.crt, tls.key, server.json
# run docker container and mount this area to /etc/web
# the default port is 8243
docker run --rm -h `hostname -f` -v /tmp/etc:/etc/web -i -t veknet/chess
```
