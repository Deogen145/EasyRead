FROM python:3.10-slim

WORKDIR /app

RUN apt-get update && apt-get install -y git libgl1 && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY clip_server.py .

EXPOSE 8001

CMD ["uvicorn", "clip_server:app", "--host", "0.0.0.0", "--port", "8001"]
