#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : doc_finder.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: 
"""

# system modules
import os
import sys
import argparse

# pymongo modules
from pymongo import MongoClient

# docx modules
import docx

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--query", action="store",
            dest="query", default="", help="Input query")
        self.parser.add_argument("--dburi", action="store",
            dest="dburi", default="", help="MongoDB URI")
        self.parser.add_argument("--dbname", action="store",
            dest="dbname", default="", help="MongoDB DB name")
        self.parser.add_argument("--dbcoll", action="store",
            dest="dbcoll", default="", help="MongoDB collection name")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def finder(text, dburi, dbname, dbcoll):
    "Simple finder of meta-data in MongoDB"
    client = MongoClient(dburi)
    coll = client[dbname][dbcoll]
    res = coll.find({'$text':{'$search': text}})
    for row in res:
        print(row['meta'])

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    finder(opts.query, opts.dburi, opts.dbname, opts.dbcoll)

if __name__ == '__main__':
    main()
