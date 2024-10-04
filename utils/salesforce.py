import csv
import os
import time

import pandas as pd
import pyarrow as pa
import pyarrow.parquet as pq
import requests
from dotenv import load_dotenv
from simple_salesforce import Salesforce


def create_folder():
    # Create a folder to store the Salesforce data
    dir_path = os.path.expanduser("~/.local/share/pipeterm_lake/salesforce")

    if not os.path.exists(dir_path):
        os.makedirs(dir_path)


def connect_to_salesforce():
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
    time.sleep(5)
    report_data = response.json()
    column_names = [
        column["label"]
        for column in report_data["reportExtendedMetadata"]["detailColumnInfo"].values()
    ]
    rows = report_data["factMap"]["T!T"]["rows"]

    csv_path = os.path.expanduser(
        f"~/.local/share/pipeterm_lake/salesforce/salesforce_report_{time.strftime('%Y%m%d%H%M%S')}.csv"
    )

    with open(csv_path, "w", newline="", encoding="UTF-8") as csvfile:
        writer = csv.writer(csvfile)
        writer.writerow(column_names)

        for row in rows:
            writer.writerow(
                [row["dataCells"][i]["label"] for i in range(len(column_names))]
            )


if __name__ == "__main__":
    create_folder()
    connect_to_salesforce()
    print(
        "Salesforce data has been downloaded and saved to the /pipeterm_lake/salesforce."
    )
