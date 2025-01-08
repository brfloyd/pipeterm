import time

import pandas as pd


def ingest_data():
    # Sample data for testing
    data = {
        "Name": ["Alice", "Bob", "Charlie"],
        "Age": [28, 34, 29],
        "Occupation": ["Engineer", "Doctor", "Artist"],
    }
    df = pd.DataFrame(data)
    time.sleep(5)  # Simulate data ingestion
    return df
