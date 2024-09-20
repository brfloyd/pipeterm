import csv
import os

import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq
import requests
from dotenv import load_dotenv
from simple_salesforce import Salesforce

# Salesforce username/password

load_dotenv()
username = os.getenv("SALESFORCE_USERNAME")
password = os.getenv("SALESFORCE_PASSWORD")
security_token = os.getenv("SALESFORCE_SECURITY_TOKEN")
instance_url = "https://zesandbox-dev-ed.develop.lightning.force.com"

sf = Salesforce(username=username, password=password, security_token=security_token)

REPORT_ID = "00OHo000002mavXMAQ"
api_version = sf.sf_version  # Getting the API version from the Salesforce instance.
report_url = f"{sf.base_url}analytics/reports/{REPORT_ID}"


print(sf.base_url)

response = requests.get(
    report_url, headers={"Authorization": f"Bearer {sf.session_id}"}
)
response.raise_for_status()

report_data = response.json()
column_names = [
    column["label"]
    for column in report_data["reportExtendedMetadata"]["detailColumnInfo"].values()
]
rows = report_data["factMap"]["T!T"]["rows"]

with open("salesforce_report.csv", "w", newline="", encoding="UTF-8") as csvfile:
    writer = csv.writer(csvfile)
    writer.writerow(column_names)

    for row in rows:
        writer.writerow(
            [row["dataCells"][i]["label"] for i in range(len(column_names))]
        )
print("Report saved as salesforce_report.csv")

df = pd.read_csv("salesforce_report.csv")
print("Report loaded into pandas dataframe")
