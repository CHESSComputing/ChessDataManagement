### Chess Data Management service

[![Build Status](https://travis-ci.org/vkuznet/ChessDataManagement.svg?branch=master)](https://travis-ci.org/vkuznet/ChessDataManagement)
[![Go CI build](https://github.com/vkuznet/ChessDataManagement/actions/workflows/go-ci.yml/badge.svg)](https://github.com/vkuznet/ChessDataManagement/actions/workflows/go-ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/vkuznet/ChessDataManagement)](https://goreportcard.com/report/github.com/vkuznet/ChessDataManagement)

### Introduction

The CHESS data flow has been discussed in this
[document](https://paper.dropbox.com/doc/HEXRD-combined-far-field-and-near-field-data-flow--Af62eKuTFDYbcbx~6Ncl4YTWAg-V4SAqod7NW6BvV6kYyTy2).

Here we propose a possible architecture for CHESS data management
based on gradual enchancement of existing infrastructure:

![ChessDataManagement](doc/images/ChessDataManagement.png)

In particular, we propose to introduce the following components:
- MetaData DB based on [MongoDB](https://www.mongodb.com) or similar
document-oriented database. Such solution should provide the following
features:
  - be able to handle free-structured text documents
  - provide reach QueryLanguage (QL)

- Files DB based on any relation database, e.g. [MySQL](https://www.mysql.com)
or free alternative [MariaDB](https://mariadb.com). The purpose of this
database is provide data bookkeeping capabilities and organize
meta-data in the following form:
  - a dataset is a collection of files (or blocks)
  - each dataset name may carry on an Experiment name and additional
  meta-data information
  - organize files in specific data-tiers, e.g. RAW for raw data,
  AOD for processed data, etc.
  - as such each dataset will have a form of a path:
    /Experiment/Processing/Tier

Both databases may reside in their own data-service called MetaData Service.
Such service can provide RESTful APIs for end-users, such as
- inject data to DBs
- fetch results
- update data in DBs
- delete data in DBs

In addition, we suggest to introduce Input Data Service which can
take care of standardization of user inputs, e.g. key-value pairs, tagging,
etc. It is not required originally, but will help in a long run to
provide uniform data representation for Meta Data Service.

Finally, the data access can be organized via XrootD service.

### References

1. [Server](web/README.md)
2. [Client](client/README.md)
3. [Maintenance](Maintenance.md)
