import pandas as pd
pd.options.mode.chained_assignment = None
import json
import argparse
from math import isnan


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



def clean_nones(value):
    """
    Recursively remove all None values from dictionaries and lists, and returns
    the result as a new dictionary or list.
    """
    if isinstance(value, list):
        return [clean_nones(x) for x in value if x is not None]
    elif isinstance(value, dict):
        return {
            key: clean_nones(val)
            for key, val in value.items()
            if val is not None
        }
    else:
        return value

def add_value_lists(initial_json, df_clean):
    df_keys_w_values = df_clean.dropna(subset=['Values']) #remove rows from dataframe without value lists
    df_idx = df_keys_w_values.index.values.tolist()
    value_lists = df_clean.iloc[:,7:].values.tolist() #hardcoded for value position column in df

    for ii in range(0,df_clean.shape[0]):
        if ii in df_idx:
            cleanedvalues = [x for x in value_lists[ii-1] if x == x] #remove nan values
            cleanedvalues.insert(0,'') #add blank field at beginning of list (per valentin request)
            initial_json[ii-1]["Values"] = cleanedvalues
        else:
            initial_json[ii-1]["Values"] = None

    return initial_json

def intermediate_dataframe(df):
    df_inter = df.iloc[:, :8] #select fields for creating json
    df_inter.rename(columns={'MetaKey':'Key'}, inplace=True) #make dictionary key "key". Doing this earlier will break pandas functions.

    return df_inter

def clean_dataframe(df):
    df_clean = df.dropna(subset=['MetaKey']).reset_index(drop=True) #remove nan keys
    df_clean.rename(columns={'MetaKey':'Key'}, inplace=True)
    df_clean = df_clean.iloc[1:,:]
    return df_clean

def create_intermediate_json(df_inter):
    json_records = df_inter.to_json(orient='records')
    json_format = json.loads(json_records)
    #print (json.dumps(json_format, indent=4))

    return json_format

def generate_schema_from_dataframe(df_clean):
    df_inter = intermediate_dataframe(df_clean)
    json_initial = create_intermediate_json(df_inter)
    json_with_values = add_value_lists(json_initial, df_clean)
    clean_json = clean_nones(json_with_values)

    return clean_json

#%%
"""
#####If not using argparse (CLI)#######

xlsx_file = 'Metadata_Schema_100322.xlsx'
beamline = 'ID1A3'
outputjson_name = 'id1a3_schema'

"""

# Make Pandas Dataframe from Beamline Specific sheet
sheet_id = beamline + '_schema'
df = pd.read_excel(xlsx_file, sheet_id)
df_clean = clean_dataframe(df)

# Generate json_schema
json_schema = generate_schema_from_dataframe(df_clean)
#print (json.dumps(clean_json, indent=4))

# Save json_schema
if outputjson_name:
    with open(outputjson_name + '.json', 'w', encoding='utf-8') as f:
        json.dump(json_schema, f, ensure_ascii=False, indent=4)
