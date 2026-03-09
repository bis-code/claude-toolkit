from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI()


class PredictRequest(BaseModel):
    text: str


class PredictResponse(BaseModel):
    prediction: float
    label: str


@app.post("/predict", response_model=PredictResponse)
async def predict(request: PredictRequest) -> PredictResponse:
    return PredictResponse(prediction=0.85, label="positive")
