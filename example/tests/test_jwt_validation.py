import os
import pytest
import requests

ENDPOINT = "http://localhost:3000"
ZITADEL_TOKEN = os.getenv("ZITADEL_TOKEN")

# Ensure token is available, otherwise skip test
@pytest.mark.skipif(ZITADEL_TOKEN is None, reason="ZITADEL_TOKEN environment variable is not set")
def test_hello_world_endpoint():
    # Define the URL of the endpoint
    url = ENDPOINT

    # Set up headers including the Authorization token
    headers = {
        "Authorization": f"Bearer {ZITADEL_TOKEN}"
    }

    # Send the GET request
    response = requests.get(url, headers=headers)

    # Assert the status code
    assert response.status_code == 200, f"Expected status code 200, but got {response.status_code}"

    # Assert the response text
    assert response.text == "Hello world", f"Expected response text 'Hello world', but got {response.text}"
