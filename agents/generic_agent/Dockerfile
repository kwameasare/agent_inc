FROM python:3.10-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
# No EXPOSE command needed, as we'll map ports dynamically.
# The CMD now runs the agent script and passes the port argument to it.
CMD ["python", "agent.py"]
