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
import json
import argparse

# pymongo modules
from pymongo import MongoClient

# docx modules
import docx

# database modules
import sqlite3

# local modules
from chess_utils import check, execute


class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--params", action="store",
            dest="params", default="", help="Input param file")
        self.parser.add_argument("--query", action="store",
            dest="query", default="", help="Input query")
        self.parser.add_argument("--list-files", action="store_true",
            dest="list_files", default=False,
            help="Provide list of files associated with meta-data")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")
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

def finder(text, params, list_files=False, verbose=False):
    "Simple finder of meta-data in MongoDB"

    # get parameters
    fname = params.get('fname')
    dburi = params.get('dburi')
    dbname = params.get('dbname')
    dbcoll = params.get('dbcoll')
    filesdb = params.get('filesdb')

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
    params = json.load(open(opts.params))
    finder(opts.query, params, opts.list_files, opts.verbose)

if __name__ == '__main__':
    main()
