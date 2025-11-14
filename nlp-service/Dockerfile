FROM python:3.11-slim AS runtime

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

COPY req.txt /tmp/req.txt
RUN pip install --upgrade pip && pip install -r /tmp/req.txt

COPY app ./app

EXPOSE 8000

ENV APP_MODULE=app.main:app
ENV APP_PORT=8000

CMD ["sh", "-c", "uvicorn ${APP_MODULE} --host 0.0.0.0 --port ${APP_PORT}"]