#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Created on Thu Oct 20 13:53:47 2022

@author: ken38
"""

import pandas as pd
pd.options.mode.chained_assignment = None
import json
import argparse
from math import isnan
import numpy as np

if __name__ == '__main__':

    # Run preprocessor
    parser = argparse.ArgumentParser(description="Create schema json from master excel spreadsheet")
    parser.add_argument('xlsx_file', metavar='xlsx', type=str, help='Path to excel file with current metadata fields')
    parser.add_argument('beamline', metavar='beamline', help='Beamline name with capitals (e.g. 1A3, 4B, etc.)', type=str)
    parser.add_argument('outputjson_name', metavar='outputjson', help='Name of json output without extension', type=str)

    args = parser.parse_args()
    xlsx_file = args.xlsx_file
    beamline = args.beamline
    outputjson_name = args.outputjson_name

def clean_dataframe(df):
    df_clean = df.dropna(subset=['metakey']).reset_index(drop=True) #remove nan keys
    df_clean.rename(columns={'metakey':'key'}, inplace=True)
    df_clean = df_clean.iloc[1:,:]
    return df_clean

def Convert(a):
    it = iter(a)
    res_dct = dict(zip(it, it))
    return res_dct
'''
#####If not using argparse (CLI)#######

xlsx_file = 'Metadata_Schema_102022.xlsx'
beamline = 'ID1A3'
outputjson_name = 'id1a3_schema'
'''
#%%
# Make Pandas Dataframe from Beamline Specific sheet
sheet_id = beamline + '_schema'
df = pd.read_excel(xlsx_file, sheet_id)
df_clean = clean_dataframe(df)


keys_list = df_clean.iloc[:,0].values.tolist()

key_dictionary = dict.fromkeys(keys_list, None)

json_format = json.dumps(key_dictionary, indent=4)
print (json_format)

# Save json_schema
if outputjson_name:
    with open(outputjson_name + '.json', 'w', encoding='utf-8') as f:
        json.dump(json_schema, f, ensure_ascii=False, indent=4)
