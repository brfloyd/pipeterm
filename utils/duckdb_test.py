import os

import duckdb

# get the path of the salesfroce directory
home_dir = os.path.expanduser("~")
salesforce_folder = os.path.join(
    home_dir, ".local", "share", "pipeterm_lake", "salesforce"
)

salesforce_sl = duckdb.read_csv(f"{salesforce_folder}/*.csv")
# Create a view so you can query it using SQL
duckdb.sql("CREATE VIEW salesforce_data AS SELECT * FROM salesforce_sl")

print(duckdb.sql("SELECT * FROM salesforce_data"))
