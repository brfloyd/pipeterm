from fastapi import APIRouter, Depends, HTTPException
from pydantic import BaseModel
from api.v1.dependencies import get_duckdb_connection

router = APIRouter()

class QueryRequest(BaseModel):
    sql: str


@router.get("/{data_lake}/tables", response_model=list[str])
def list_tables(data_lake: str, conn=Depends(get_duckdb_connection)):

    try:

        tables = conn.execute("SHOW TABLES").fetchall()
        return [table[0] for table in tables]
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/{data_lake}/query")
def execute_query(data_lake: str, query_request: QueryRequest, conn=Depends(get_duckdb_connection)):
    try:
        result = conn.execute(query_request.sql).fetchall()
        columns = [desc[0] for desc in conn.description]
        formatted_result = [dict(zip(columns, row)) for row in result]
        return {"columns": columns, "rows": formatted_result}
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Query failed: {str(e)}")


@router.get("/{data_lake}/files", response_model=list[str])
def list_files(data_lake: str):
    """
    List all CSV files available in the specified data lake directory.
    """
    import os
    from api.v1.dependencies import BASE_DATALAKE_DIR

    data_lake_path = os.path.join(BASE_DATALAKE_DIR, data_lake)


    if not os.path.isdir(data_lake_path):
        raise HTTPException(status_code=404, detail=f"Data lake '{data_lake}' does not exist")

    try:
        files = [f for f in os.listdir(data_lake_path) if f.endswith(".csv")]
        return files
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
