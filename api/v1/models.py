from pydantic import BaseModel

class QueryRequest(BaseModel):
    sql: str

class QueryResponse(BaseModel):
    columns: list[str]
    rows: list[str]
