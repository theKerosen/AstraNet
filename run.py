import requests
import os
import subprocess
import threading
import time


class run_api:
    def setup(self):
        os.environ["PYTHONUNBUFFERED"] = "1"

        node = subprocess.Popen(
            ["node", f"{os.getcwd()}/rest_engine/dist/src/server.js"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            bufsize=1,
        )
        threading.Thread(target=self.read_output, args=(node, "Neon")).start()

        python = subprocess.Popen(
            ["python", f"{os.getcwd()}/astra/src/main.py"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            universal_newlines=True,
            bufsize=1,
        )
        threading.Thread(target=self.read_output, args=(python, "Astra")).start()
        while True:
            try:
                connection = requests.get("http://localhost:3000/app/changelist")
                if connection.status_code == 200:
                    break
            except:
                time.sleep(0.1)
            finally:
                time.sleep(3)
                rust = subprocess.Popen(
                    [f"{os.getcwd()}/data_engine/bin/data_engine.exe"],
                    stdin=subprocess.PIPE,
                )
                rust.stdin.write(b"730")

    def read_output(self, process, process_name):
        for line in iter(process.stdout.readline, b""):
            if line:
                print(f"[{process_name} I/O] {line.strip()}", flush=True)
        for line in iter(process.stderr.readline, b""):
            if line:
                print(f"[{process_name} Error] {line.strip()}", flush=True)


run_api().setup()
