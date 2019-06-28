#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : chess_finder.py
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

# database modules
import sqlite3

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--query", action="store",
            dest="query", default="", help="Input query")
        self.parser.add_argument("--dburi", action="store",
            dest="dburi", default="mongodb://localhost:8230",
            help="MongoDB URI, default mongodb://localhost:8230")
        self.parser.add_argument("--dbname", action="store",
            dest="dbname", default="chess",
            help="MongoDB DB name, default chess")
        self.parser.add_argument("--dbcoll", action="store",
            dest="dbcoll", default="meta",
            help="MongoDB collection name, default meta")
        self.parser.add_argument("--filesdb", action="store",
            dest="filesdb", default="files.db",
            help="FilesDB URI, default files.db")
        self.parser.add_argument("--list-files", action="store_true",
            dest="list_files", default=False,
            help="Provide list of files associated with meta-data")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def execute(cur, stmt, bindings, verbose=None):
    "Helper function to execute statement"
    if verbose:
        print(stmt)
        print(bindings)
    cur.execute(stmt, bindings)
    rows = cur.fetchall()
    for row in rows:
        return row[0]

def find_files(filesdb, dataset, verbose=False):
    "Find associated files for a given dataset"
    # open DB connection
    conn = sqlite3.connect(filesdb)
    _, experiment, processing, tier = dataset.split('/')

    # get experiment, processing, tier IDs
    cur = conn.cursor()
    stmt = 'SELECT experiment_id FROM experiments WHERE name=?'
    eid = execute(cur, stmt, (experiment,), verbose)
    stmt = 'SELECT processing_id FROM processing WHERE name=?'
    pid = execute(cur, stmt, (processing,), verbose)
    stmt = 'SELECT tier_id FROM tiers WHERE name=?'
    tid = execute(cur, stmt, (tier,), verbose)

    # find dataset id
    stmt = 'SELECT dataset_id FROM datasets WHERE experiment_id=? and processing_id=? and tier_id=?'
    did = execute(cur, stmt, (eid, pid, tid), verbose)
    if verbose:
        print("dataset {}, did {}".format(dataset, did))
    if not did:
        return
    stmt = 'SELECT name FROM files WHERE dataset_id=?'
    cur.execute(stmt, (did,))
    for row in cur.fetchall():
        yield row[0]

def finder(text, dburi, dbname, dbcoll, filesdb=None, list_files=False, verbose=False):
    "Simple finder of meta-data in MongoDB"
    client = MongoClient(dburi)
    coll = client[dbname][dbcoll]
    res = coll.find({'$text':{'$search': text}})
    for row in res:
        print(row['meta'])
        if list_files:
            dataset = row['dataset']
            print('Associated dataset {}'.format(dataset))
            print('Associated files:')
            files = find_files(filesdb, dataset, verbose)
            for fname in files:
                print(fname)

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    finder(opts.query, opts.dburi, opts.dbname, opts.dbcoll, opts.filesdb, opts.list_files, opts.verbose)

if __name__ == '__main__':
    main()
