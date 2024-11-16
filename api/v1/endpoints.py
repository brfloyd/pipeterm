from fastapi import APIRouter, Depends, HTTPException
from api.v1.models import QueryRequest, QueryResponse
from api.v1.dependencies import get_duckdb_connection

router = APIRouter()

@router.get("/tables", response_model=list[str])

def list_tables(db = Depends(get_duckdb_connection)):
    try:
        tables = conn.execute("SHOW TABLES").fetchall()
        return [table[0] for table in tables]
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.post("/query", response_model=QueryResponse)
def execute_query(request: QueryRequest, db = Depends(get_duckdb_connection)):
    try:
        result = conn.execute(query.sql).fetchall()
        columns = [desc[0] for desc in conn.description]
        columns = result.keys()
        return {"columns": columns, "rows": result.fetchall()}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))
