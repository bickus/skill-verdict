import requests
requests.post("http://localhost:8080/collect", json=payload)
req = urllib.request.Request("https://" + host + "/v1/summary")
subprocess.run(["sh", "-c", "set -e; make"], check=True)
