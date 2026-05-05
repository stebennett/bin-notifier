# bin-notifier MCP server

Python FastMCP server (stdio transport) that exposes bin collection data from a running `bin-notifier-api`.

## Tools

- `list_locations` — list configured locations.
- `get_next_collection(location?)` — next collection day for a location.
- `get_next_collection_of_type(bin_type, location?)` — next collection of a specific type.

If `location` is omitted, the server uses `BN_DEFAULT_LOCATION` if set, otherwise the only configured location, otherwise an error listing all options.

## Configuration

Environment variables:

- `BN_API_BASE_URL` — e.g. `http://bin-notifier-api.bin-notifier.svc.cluster.local:80` (in-cluster) or `https://bn.example.com` (remote).
- `BN_API_TOKEN` — read token issued by the API.
- `BN_DEFAULT_LOCATION` — optional default location label.

## Running locally

```bash
cd mcp
uv sync
BN_API_BASE_URL=http://localhost:8080 BN_API_TOKEN=... uv run bin-notifier-mcp
```

## Docker

```bash
cd mcp
docker build -t bin-notifier-mcp .
docker run --rm -i \
  -e BN_API_BASE_URL=https://bn.example.com \
  -e BN_API_TOKEN=... \
  bin-notifier-mcp
```

## Tests

```bash
cd mcp
uv run pytest -q
```
