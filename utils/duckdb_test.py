import duckdb

duckdb.read_csv("salesforce_report.csv")
print(duckdb.sql("SELECT * FROM salesforce_report.csv"))
