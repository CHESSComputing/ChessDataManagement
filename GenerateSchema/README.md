In an environment with:
python >= 3.8.3
pandas >= 1.0.5

From CLI:

Python generate_schema.py (.xlsx file) (beamline) (schema name)

Where:
.xlsx file is the Master Excel file from SharePoint
Beamline is written with ID prefix (eg ID1A3, ID4B)
schema output is file name without extension (no spaces, start with letter)


This will generate a json in the folder the program is executed in.
Alternatively you can provide a file path in front of the output file name just do not include the .json extension. 
