import os
import pytest
import requests

ENDPOINT = "http://localhost:3000"
ZITADEL_ID_TOKEN = os.getenv("ZITADEL_ID_TOKEN")
ZITADEL_ACCESS_TOKEN = os.getenv("ZITADEL_ACCESS_TOKEN")
# Backward-compatible: if only ZITADEL_TOKEN is set, use it as access token
ZITADEL_TOKEN = os.getenv("ZITADEL_TOKEN")
if ZITADEL_ACCESS_TOKEN is None:
    ZITADEL_ACCESS_TOKEN = ZITADEL_TOKEN


def _get(token: str) -> requests.Response:
    return requests.get(ENDPOINT, headers={"Authorization": f"Bearer {token}"})


@pytest.mark.skipif(ZITADEL_ID_TOKEN is None, reason="ZITADEL_ID_TOKEN environment variable is not set")
def test_id_token():
    response = _get(ZITADEL_ID_TOKEN)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    assert response.text == "Hello world", f"Expected 'Hello world', got {response.text}"


@pytest.mark.skipif(ZITADEL_ACCESS_TOKEN is None, reason="ZITADEL_ACCESS_TOKEN environment variable is not set")
def test_access_token():
    response = _get(ZITADEL_ACCESS_TOKEN)
    assert response.status_code == 200, f"Expected 200, got {response.status_code}"
    assert response.text == "Hello world", f"Expected 'Hello world', got {response.text}"


def test_missing_token():
    response = requests.get(ENDPOINT)
    assert response.status_code == 401, f"Expected 401, got {response.status_code}"


def test_invalid_token():
    response = _get("invalid.token.value")
    assert response.status_code == 401, f"Expected 401, got {response.status_code}"


def test_malformed_header():
    response = requests.get(ENDPOINT, headers={"Authorization": "NotBearer sometoken"})
    assert response.status_code == 401, f"Expected 401, got {response.status_code}"
