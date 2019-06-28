#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : doc_parser.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: 
"""

# system modules
import os
import sys
import json
import time
import argparse

# pymongo modules
from pymongo import MongoClient

# docx modules
import docx

# sqlite3 module
import sqlite3

# local modules
from chess_utils import check, execute


class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--params", action="store",
            dest="params", default="", help="Input param file")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def files(path):
    "return list of files in a given path (experiment)"
    # python 2
    result = [os.path.join(dp, f) for dp, dn, filenames in os.walk(path) for f in filenames]
    # python 3
#     result = glob.glob(path + '/*', recursive=True)
    return result

def files_parser(filesdb, path, dataset, verbose=None):
    "filesdb parser, insert files from a given path to DB under given dataset"
    # open DB connection
    conn = sqlite3.connect(filesdb)
    _, experiment, processing, tier = dataset.split('/')

    # insert experiment, processing, tier data
    tstamp = int(time.time())
    statements = [
    "INSERT INTO tiers (name) VALUES ('{}')".format(tier),
    "INSERT INTO processing (name) VALUES ('{}')".format(processing),
    "INSERT INTO experiments (name) VALUES ('{}')".format(experiment),
            ]
    for stmt in statements:
        if verbose:
            print(stmt)
        conn.execute(stmt)

    # get experiment, processing, tier IDs
    cur = conn.cursor()
    stmt = 'SELECT experiment_id FROM experiments WHERE name=?'
    eid = execute(cur, stmt, (experiment,), verbose)
    stmt = 'SELECT processing_id FROM processing WHERE name=?'
    pid = execute(cur, stmt, (processing,), verbose)
    stmt = 'SELECT tier_id FROM tiers WHERE name=?'
    tid = execute(cur, stmt, (tier,), verbose)

    # insert data into datasets table
    stmt = "INSERT INTO datasets (experiment_id,processing_id,tier_id,tstamp) \
      VALUES ('{}', '{}', '{}', {} )".format(eid, pid, tid, tstamp)
    conn.execute(stmt)

    # find out dataset id
    stmt = 'SELECT dataset_id FROM datasets WHERE experiment_id=? and processing_id=? and tier_id=?'
    did = execute(cur, stmt, (eid, pid, tid), verbose)
    if verbose:
        print("eid {} pid {} tid {} did {}".format(eid, pid, tid, did))

    # insert files info
    for name in files(path):
        conn.execute("INSERT INTO FILES (dataset_id,name) VALUES ({},'{}')".format(did, name));

    # commit all records
    conn.commit()

    # close connection
    conn.close()

    return did

def parser(params, verbose=None):
    "Simple parser to insert CHESS document into MongoDB"
    check(params)
    if verbose:
        print('Input parameters: {}'.format(json.dumps(params)))

    # get parameters
    fname = params.get('fname')
    path = params.get('path')
    dburi = params.get('dburi')
    dbname = params.get('dbname')
    dbcoll = params.get('dbcoll')
    filesdb = params.get('filesdb')
    experiment = params.get('experiment')
    processing = params.get('processing')
    tier = params.get('tier')

    if verbose:
        print('Parse {}'.format(fname))
    doc = docx.Document(fname)
    text = ""
    for para in doc.paragraphs:
        text += para.text
    if not text:
        return
    client = MongoClient(dburi)
    coll = client[dbname][dbcoll]
    # create a dataset and insert all files for it
    dataset = '/{}/{}/{}'.format(experiment, processing, tier)
    if verbose:
        print('Insert files for {} dataset'.format(dataset))
    did = files_parser(filesdb, path, dataset, verbose)
    # prepare meta data info and insert it into MetaData DB
    if verbose:
        print('Insert meta data "{}" for dataset {}'.format(text.encode('utf-8'), did))
    res = coll.insert_one({'meta':text, 'dataset':dataset, 'did':did})
#     metaId = '{}'.format(res.inserted_id)
#     for name in files(path):
#         coll.insert_one({'metaId': metaId, 'name': name})

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    params = json.load(open(opts.params))
    parser(params, opts.verbose)

if __name__ == '__main__':
    main()
