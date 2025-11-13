FROM python:3.11-slim AS base

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY req.txt .
RUN pip install --upgrade pip && pip install -r req.txt

COPY app ./app

EXPOSE 8081

ENV TUNA_AUTOSTART=false
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8081"]