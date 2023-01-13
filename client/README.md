This area contains CHESS CLI codebase. It can be compiled using the following
command:
```
go build
```
and, then we can run it as following:
```
./chess_client -help

Command line interface to CHESS Data Management System

Obtain kerberos ticket:
kinit -c krb5_ccache <username>

Options:
  -did int
    	show files for given dataset-id
  -insert string
    	insert record to the server
  -krbFile string
    	kerberos file
  -query string
    	query string to look-up your data
  -schema string
    	schema name for your data
  -uri string
    	CHESS Data Management System URI (default "https://chessdata.classe.cornell.edu:8243")
  -verbose int
    	verbosity level

Examples:

# inject new record into the system using lite schema
chess_client -krbFile krb5cc_ccache -insert record.json -schema lite

# look-up data from the system using free text-search
chess_client -krbFile krb5cc_ccache -query="search words"

# look-up data from the system using keyword search
chess_client -krbFile krb5cc_ccache -query="proposal:123"

# look-up files for specific dataset-id
chess_client -krbFile krb5cc_ccache -did=1570563920579312510
```
