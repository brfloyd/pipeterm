from fastapi import FastAPI
from api.v1.endpoints import router as api_router

app = FastAPI(
    title="Pipeterm API",
    description="API for Accesing Pipeterm Datalake",
    version="1.0.0",
)

app.include_router(api_router, prefix="/api/v1")


@app.get("/")
def read_root():
    return {"message": "pipeterm API is running"}
