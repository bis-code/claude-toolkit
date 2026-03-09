from fastapi.testclient import TestClient
from main import app

client = TestClient(app)


def test_predict_returns_positive_label():
    response = client.post("/predict", json={"text": "great product"})
    assert response.status_code == 200
    body = response.json()
    assert body["label"] == "positive"
    assert body["prediction"] == 0.85
