## Run the API for Testing

Must be done in the pipeterm root directory.
```
uvicorn api.main:app --reload --app-dir .

```

## Test inputs for API

Show tables
```
curl -X GET http://127.0.0.1:8000/api/v1/salesforce/tables

```

Query a a know table
```
curl -X POST http://127.0.0.1:8000/api/v1/salesforce/query \
     -H "Content-Type: application/json" \
     -d '{"sql": "SELECT * FROM salesforce_report_20240101 LIMIT 5"}'

```
