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
import argparse

# pymongo modules
from pymongo import MongoClient

# docx modules
import docx

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input meta-data file")
        self.parser.add_argument("--path", action="store",
            dest="path", default="", help="Input path of experiment")
        self.parser.add_argument("--dburi", action="store",
            dest="dburi", default="", help="MongoDB URI")
        self.parser.add_argument("--dbname", action="store",
            dest="dbname", default="", help="MongoDB DB name")
        self.parser.add_argument("--dbcoll", action="store",
            dest="dbcoll", default="", help="MongoDB collection name")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def files(path):
    "return list of files in a given path (experiment)"
    # python 2
    result = [os.path.join(dp, f) for dp, dn, filenames in os.walk(path) for f in filenames]
    # python 3
#     result = glob.glob(path + '/*', recursive=True)
    return result

def parser(fname, path, dburi, dbname, dbcoll):
    "Simple parser to insert CHESS document into MongoDB"
    doc = docx.Document(fname)
    text = ""
    for para in doc.paragraphs:
        text += para.text
    if not text:
        return
    client = MongoClient(dburi)
    coll = client[dbname][dbcoll]
    res = coll.insert_one({'meta':text})
    metaId = '{}'.format(res.inserted_id)
    for name in files(path):
        coll.insert_one({'metaId': metaId, 'name': name})

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    parser(opts.fin, opts.path, opts.dburi, opts.dbname, opts.dbcoll)

if __name__ == '__main__':
    main()
