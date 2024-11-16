import os
import duckdb
from fastapi import HTTPException

BASE_DATALAKE_DIR = os.path.join(os.path.expanduser("~"), ".local", "share", "pipeterm_lake")

def get_duckdb_connection(data_lake: str):
    data_lake_path = os.path.join(BASE_DATALAKE_DIR, data_lake)
    if not os.path.isdir(data_lake_path):
        raise HTTPException(status_code=404, detail=f"Data lake '{data_lake}' does not exist")

    # Create an in-memory DuckDB connection
    conn = duckdb.connect()

    for file_name in os.listdir(data_lake_path):
        if file_name.endswith(".csv"):
            table_name = os.path.splitext(file_name)[0]
            file_path = os.path.join(data_lake_path, file_name)
            create_view_query = f"CREATE VIEW {table_name} AS SELECT * FROM read_csv_auto('{file_path}')"
            conn.execute(create_view_query)

    ##This is where the connection is reuturned and closed after use
    try:
        yield conn
    finally:
        conn.close()
