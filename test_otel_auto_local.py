import requests

def make_requests():
    url = "https://httpbin.org/get"  # Public API for testing
    for i in range(5):
        response = requests.get(url)
        print(f"Request {i+1}: {response.status_code} - {response.json()}")

if __name__ == "__main__":
    print("Sending requests to a public API...")
    make_requests()
    print("Done! Check Jaeger or Grafana Cloud for traces.")
