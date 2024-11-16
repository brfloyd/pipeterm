import duckdb
from fastapi import Depends

def get_duckdb_connection():
    conn = duckdb.connect("pipeterm.db")
    try :
        yield conn
    finally:
        conn.close()
