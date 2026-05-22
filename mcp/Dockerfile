# syntax=docker/dockerfile:1.7
FROM python:3.12-slim AS build
WORKDIR /app
ENV UV_LINK_MODE=copy
COPY --from=ghcr.io/astral-sh/uv:0.5.0 /uv /usr/local/bin/uv
COPY pyproject.toml uv.lock ./
COPY src ./src
RUN uv sync --frozen --no-dev
RUN uv build --wheel

FROM python:3.12-slim
WORKDIR /app
COPY --from=build /app/dist/*.whl /tmp/
RUN pip install --no-cache-dir /tmp/*.whl && rm /tmp/*.whl
USER 1000:1000
ENTRYPOINT ["bin-notifier-mcp"]
