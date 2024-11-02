import importlib.util
import os
import sys
from datetime import datetime

import pandas as pd

script_path = sys.argv[1]


def verify_path(script_path):
    if not os.path.isfile(script_path):
        print(f"Error: {script_path} not found.")
        sys.exit(1)


def load_script(script_path):
    try:
        spec = importlib.util.spec_from_file_location("script", script_path)
        script = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(script)
        return script
    except Exception as e:
        print(f"Error loading script: {e}")
        sys.exit(1)


def check_for_ingest_data_function(script_path):
    script = load_script(script_path)
    if not hasattr(script, "ingest_data"):
        print("Error: ingest_data function not found in script.")
        sys.exit(1)


def ingest_data(user_script):
    try:
        df = load_script(script_path).ingest_data()
    except Exception as e:
        print(f"Error executing script, check script rules: {e}")
        sys.exit(1)
    if not isinstance(df, pd.DataFrame):
        print("Error: ingest_data function must return a pandas DataFrame.")
        sys.exit(1)
    return df


def save_data(df, output_dir):
    home_dir = os.path.expanduser("~")
    output_dir = os.path.join(home_dir, ".local", "share", "pipeterm_lake")
    os.makedirs(output_dir, exist_ok=True)

    timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    script_name = os.path.splitext(os.path.basename(script_path))[0]
    output_file = f"{script_name}_{timestamp}.csv"
    output_path = os.path.join(output_dir, output_file)

    try:
        df.to_csv(output_path, index=False)
        print(f"Data saved to: {output_path}")
    except Exception as e:
        print(f"Error saving data: {e}")
        sys.exit(1)


if __name__ == "__main__":
    verify_path(script_path)
    check_for_ingest_data_function(script_path)
    df = ingest_data(script_path)
    save_data(df, script_path)
