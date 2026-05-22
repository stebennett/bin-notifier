from __future__ import annotations

import os
from datetime import date, datetime
from typing import Any
from zoneinfo import ZoneInfo

from mcp.server.fastmcp import FastMCP

from .client import BinNotifierClient, NoData, NoMatch, UnknownLocation

mcp = FastMCP("bin-notifier")

_LONDON = ZoneInfo("Europe/London")


def _make_client() -> BinNotifierClient:
    base = os.environ.get("BN_API_BASE_URL")
    token = os.environ.get("BN_API_TOKEN")
    if not base or not token:
        raise RuntimeError("BN_API_BASE_URL and BN_API_TOKEN must be set")
    return BinNotifierClient(base, token)


async def _resolve_location(client: BinNotifierClient, label: str | None) -> str | dict[str, str]:
    if label:
        return label
    default = os.environ.get("BN_DEFAULT_LOCATION")
    if default:
        return default
    locs = await client.list_locations()
    if len(locs) == 1:
        return locs[0]["label"]
    labels = ", ".join(l["label"] for l in locs)
    return {"error": f"location is required (configured: {labels})"}


def _days_until(target: str) -> int:
    today = datetime.now(_LONDON).date()
    return (date.fromisoformat(target) - today).days


async def _call_and_annotate(coro: Any, label: str, bin_type: str | None = None) -> dict[str, Any]:
    """Await a get_next_collection call and normalise its result.

    Shapes returned to MCP consumers:
      - {"error": ...}                      -> bad input or unavailable cache
      - {"status": "none_scheduled", ...}   -> valid query, nothing upcoming
      - the API payload + "days_until"      -> a hit
    """
    try:
        resp = await coro
    except UnknownLocation:
        return {"error": f"unknown location: {label}"}
    except NoData:
        return {"error": f"no data cached yet for {label}"}
    except NoMatch:
        type_phrase = f" {bin_type}" if bin_type else ""
        return {
            "status": "none_scheduled",
            "location": label,
            "message": f"no upcoming{type_phrase} collection found for {label}",
        }
    resp["days_until"] = _days_until(resp["date"])
    return resp


@mcp.tool(name="list_locations", description="List configured bin-notifier locations.")
async def list_locations_tool() -> list[dict[str, str]]:
    return await _make_client().list_locations()


@mcp.tool(
    name="get_next_collection",
    description="Return the next bin collection day for a location. Omit `location` to use the default.",
)
async def get_next_collection_tool(location: str | None = None) -> dict[str, Any]:
    client = _make_client()
    label = await _resolve_location(client, location)
    if isinstance(label, dict):
        return label
    return await _call_and_annotate(client.get_next_collection(label), label)


@mcp.tool(
    name="get_next_collection_of_type",
    description="Return the next collection of a specific bin type (e.g. 'Food Waste').",
)
async def get_next_collection_of_type_tool(bin_type: str, location: str | None = None) -> dict[str, Any]:
    client = _make_client()
    label = await _resolve_location(client, location)
    if isinstance(label, dict):
        return label
    return await _call_and_annotate(
        client.get_next_collection(label, bin_type=bin_type), label, bin_type
    )


def run() -> None:
    mcp.run()
