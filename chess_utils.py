def check(params):
    "Check input parameters"
    attrs = []
    for attr in attrs:
        if attr not in params:
            raise Exception('key {} not in {}'.format(attr, json.dumps(params)))

def execute(cur, stmt, bindings, verbose=None):
    "Helper function to execute statement"
    if verbose:
        print(stmt)
        print(bindings)
    cur.execute(stmt, bindings)
    rows = cur.fetchall()
    for row in rows:
        return row[0]
