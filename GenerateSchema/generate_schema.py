

import pandas as pd
pd.options.mode.chained_assignment = None
import json
import argparse


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

# %% Definitions
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
    newkey, excelkey = convert_to_schema_dict(7)
    df_keys_w_values = df_clean.dropna(subset=[excelkey]) #remove rows from dataframe without value lists
    df_keys_w_values.rename(columns={excelkey:newkey})
    #df_keys_w_values = df_clean.dropna(subset=['values']) #remove rows from dataframe without value lists
    #df_keys_w_values.rename(columns={'values':'value'})
    df_idx = df_keys_w_values.index.values.tolist()
    value_lists = df_clean.iloc[:,7:].values.tolist() #hardcoded for value position column in df

    for ii in range(0,df_clean.shape[0]+1):
        if ii in df_idx:
            cleanedvalues = [x for x in value_lists[ii-1] if x == x] #remove nan values
            cleanedvalues.insert(0,'') #add blank field at beginning of list (per valentin request)
            initial_json[ii-1][newkey] = cleanedvalues
        else:
            initial_json[ii-1][newkey] = None

    return initial_json


def intermediate_dataframe(df):
    df_inter = df.iloc[:, :8] #select fields for creating json
    newkey, excelkey = convert_to_schema_dict(0)
    df_inter.rename(columns={excelkey:newkey}, inplace=True) #make dictionary key "key". Doing this earlier will break pandas functions.

    return df_inter


def clean_dataframe(df):
    newkey, excelkey = convert_to_schema_dict(0) #key from dictionary 
    df_clean = df.dropna(subset=[excelkey]).reset_index(drop=True) #remove nan key
    df_clean.rename(columns={excelkey:newkey}, inplace=True)
    df_clean = df_clean.iloc[1:,:]
    return df_clean


def create_intermediate_json(df_inter):
    newkey, excelkey = convert_to_schema_dict(7) # 7 index in excel dataframe for "value" 
    #df_inter.rename(columns={'values':'value'}, inplace=True)
    df_inter.rename(columns={excelkey:newkey}, inplace=True)
    json_records = df_inter.to_json(orient='records')
    json_format = json.loads(json_records)
    #print (json.dumps(json_format, indent=4))

    return json_format


def generate_schema_from_dataframe(df_clean):
    df_clean = check_dataypes(df_clean)
    df_inter = intermediate_dataframe(df_clean)
    json_initial = create_intermediate_json(df_inter)
    json_with_values = add_value_lists(json_initial, df_clean)
    clean_json = clean_nones(json_with_values)

    return clean_json


def check_dataypes(df_clean):
    
    nanidx_placeholder = df_clean.loc[pd.isna(df_clean["placeholder"]), :].index
    nanindx_description = df_clean.loc[pd.isna(df_clean["description"]), :].index
    
    for x in range(1, df_clean.shape[0]):
        
        if x not in nanidx_placeholder:
            df_clean['placeholder'][x] = str(df_clean['placeholder'][x])
        else:
            pass
        if x not in nanindx_description:
            df_clean['description'][x] = str(df_clean['description'][x])
        else:
            pass
    return df_clean

def convert_to_schema_dict(idx): 
    exceldict_df = pd.read_excel(xlsx_file, 'schema_dicts') #pull from dict in excel workbook
    exceldict_df = exceldict_df.iloc[:,:2]
    newkey = exceldict_df['ServiceDict'][idx]
    excelkey = exceldict_df['ExcelDict'][idx]
    
    return newkey, excelkey

#%%
'''
#####If not using argparse (CLI)#######

xlsx_file = 'metadata_excel_102022.xlsx'
beamline = 'ID1A3'
outputjson_name = 'BeamlineSchema/id1a3_new_schema'

'''

#%%
# Make Pandas Dataframe from Beamline Specific sheet
sheet_id = beamline + '_schema'
df = pd.read_excel(xlsx_file, sheet_id)
df_clean = clean_dataframe(df)
#%%
# Generate json_schema
json_schema = generate_schema_from_dataframe(df_clean)
print (json.dumps(json_schema, indent=4))
#%%
# Save json_schema
if outputjson_name:
    with open(outputjson_name + '.json', 'w', encoding='utf-8') as f:
        json.dump(json_schema, f, ensure_ascii=False, indent=4)
else: 
    with open(beamline + '.json', 'w', encoding='utf-8') as f:
        json.dump(json_schema, f, ensure_ascii=False, indent=4)