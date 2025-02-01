import requests
import concurrent.futures
import time
import random

# Configuration
API_URL = "http://localhost:3001/v1/ask"
# API_URL = "https://thewriterco-auth-go.onrender.com/v1/ask"  # Replace with your API endpoint
TOTAL_REQUESTS = 20  # Total requests to send
CONCURRENT_REQUESTS = 1  # Number of concurrent threads
MAX_RETRIES = 3  # Number of times to retry a request if rate-limited

# Sample request payload (adjust according to your API needs)
payload = "{\"message\":{\"conversation\":\"koKmc2VuZGVyokFJp2NvbnRlbnTZKEhpISDwn5mPIEhvdyBtYXkgSSBiZSBvZiBzZXJ2aWNlIHRvIHlvdT+CpnNlbmRlcqRVc2Vyp2NvbnRlbnSpSGVsbG8gQUkh\",\"additionalContext\":[\"-@here Note user's theme for appropriate styling: dark mode\",\"-@here Note user's current book state for your knowledge: Bible 1:1\",\"-@here Note user's local time for context: 1/31/2025, 2:59:49 PM\",\"-@here Note user's current date for context: Fri Jan 31 2025\"]}}"

# headers = {"Content-Type": "application/x-www-form-urlencoded"}

# Function to send a request with retry mechanism for 429 errors
def send_request(session, request_id):
    retries = 0
    while retries <= MAX_RETRIES:
        try:
            start_time = time.time()
            response = session.post(API_URL, data=payload)
            end_time = time.time()

            latency = end_time - start_time
            response_text = response.text

            if response.status_code == 200:
                return (request_id, response.status_code, latency, response_text)

            elif response.status_code == 429:
                retry_delay = 2 ** retries + random.uniform(0, 0.5)  # Exponential backoff with jitter
                time.sleep(1)  # Wait for 30 seconds before retrying
                print(f"⚠️ Request {request_id} rate limited (429). Retrying in {retry_delay:.2f} seconds...")
                time.sleep(1)
                retries += 1
            else:
                return (request_id, response.status_code, latency, response_text)

        except requests.exceptions.RequestException as e:
            return (request_id, "FAILED", 0, str(e))

    return (request_id, 429, 0, "Rate limit exceeded after retries")

# Run concurrent requests
def stress_test():
    success_count = 0
    failure_count = 0
    rate_limited_count = 0
    total_latency = 0.0
    latencies = []

    with requests.Session() as session, concurrent.futures.ThreadPoolExecutor(max_workers=CONCURRENT_REQUESTS) as executor:
        futures = {executor.submit(send_request, session, i): i for i in range(TOTAL_REQUESTS)}

        for future in concurrent.futures.as_completed(futures):
            result = future.result()
            request_id = result[0]

            if result[1] == 200:
                success_count += 1
                latencies.append(result[2])
                total_latency += result[2]
                print(f"✅ Request {request_id} succeeded in {result[2]:.2f} seconds")
                print(f"🔹 Response: {result[3]}")  # Print response text
            elif result[1] == 429:
                rate_limited_count += 1
                print(f"🚫 Request {request_id} exceeded rate limit after retries.")
            else:
                failure_count += 1
                print(f"❌ Request {request_id} failed ({result[1]}): {result[3] if len(result) > 3 else 'Unknown error'}")

    # Print test summary
    avg_latency = (total_latency / success_count) if success_count else 0
    max_latency = max(latencies) if latencies else 0
    min_latency = min(latencies) if latencies else 0

    print("\n===== Stress Test Summary =====")
    print(f"Total Requests: {TOTAL_REQUESTS}")
    print(f"Successful: {success_count}")
    print(f"Rate-Limited (429): {rate_limited_count}")
    print(f"Failed: {failure_count}")
    print(f"Average Latency: {avg_latency:.2f} seconds")
    print(f"Max Latency: {max_latency:.2f} seconds")
    print(f"Min Latency: {min_latency:.2f} seconds")

if __name__ == "__main__":
    stress_test()
