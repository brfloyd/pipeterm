import os
import platform


def get_lake_folder():
    home_dir = os.path.expanduser("~")

    data_lake_folder = os.path.join(home_dir, ".local", "share", "pipeterm_lake")
    print("Attempting to create data lake folder at: ")

    if not os.path.exists(data_lake_folder):
        os.makedirs(data_lake_folder)
        print("Data lake folder created at: ")

    return data_lake_folder


if __name__ == "__main__":
    get_lake_folder()
