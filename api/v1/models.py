from pydantic import BaseModel

class QueryRequest(BaseModel):
    sql: str

class QueryResponse(BaseModel):
    columns: List[str]
    rows: List[str]
