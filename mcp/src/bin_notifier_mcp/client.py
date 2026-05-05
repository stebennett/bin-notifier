from __future__ import annotations

from typing import Any
from urllib.parse import quote

import httpx


class ApiError(RuntimeError):
    """Generic API error."""


class NotFound(ApiError):
    """404 from the API (unknown location or no matching collection)."""


class NoData(ApiError):
    """503 from the API (location known but no cache yet)."""


class Unauthorized(ApiError):
    """401 from the API."""


class BinNotifierClient:
    def __init__(self, base_url: str, token: str, *, timeout: float = 5.0) -> None:
        self._base_url = base_url.rstrip("/")
        self._token = token
        self._timeout = timeout

    def _headers(self) -> dict[str, str]:
        return {"Authorization": f"Bearer {self._token}"}

    async def _get(self, path: str, params: dict[str, Any] | None = None) -> Any:
        async with httpx.AsyncClient(timeout=self._timeout) as c:
            try:
                resp = await c.get(self._base_url + path, params=params, headers=self._headers())
            except httpx.HTTPError as e:
                raise ApiError(f"network error contacting bin-notifier API: {e}") from e

        if resp.status_code == 401:
            raise Unauthorized("unauthorized")
        if resp.status_code == 404:
            raise NotFound(_msg(resp))
        if resp.status_code == 503:
            raise NoData(_msg(resp))
        if resp.status_code >= 400:
            raise ApiError(f"API returned {resp.status_code}: {resp.text}")
        return resp.json()

    async def list_locations(self) -> list[dict[str, str]]:
        return await self._get("/v1/locations")

    async def list_collections(
        self, label: str, *, from_date: str | None = None, bin_types: list[str] | None = None
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if from_date:
            params["from"] = from_date
        if bin_types:
            params["type"] = bin_types
        return await self._get(f"/v1/locations/{quote(label, safe='')}/collections", params=params or None)

    async def get_next_collection(
        self, label: str, *, bin_type: str | None = None, from_date: str | None = None
    ) -> dict[str, Any]:
        params: dict[str, Any] = {}
        if from_date:
            params["from"] = from_date
        if bin_type:
            params["type"] = bin_type
        return await self._get(f"/v1/locations/{quote(label, safe='')}/collections/next", params=params or None)


def _msg(resp: httpx.Response) -> str:
    try:
        return str(resp.json().get("error") or resp.text)
    except ValueError:
        return resp.text
