import pytest
import httpx
import respx

from bin_notifier_mcp.client import BinNotifierClient, ApiError, NotFound, NoData, UnknownLocation, NoMatch


@pytest.mark.asyncio
async def test_list_locations_returns_parsed_json():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations").mock(
            return_value=httpx.Response(200, json=[{"label": "Home", "postcode": "RG12"}])
        )
        client = BinNotifierClient("https://api.example", "tok")
        got = await client.list_locations()
        assert got == [{"label": "Home", "postcode": "RG12"}]


@pytest.mark.asyncio
async def test_get_next_collection_passes_type_filter_and_token():
    async with respx.mock(base_url="https://api.example") as mock:
        route = mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(200, json={
                "location": "Home", "scraped_at": "2026-05-05T18:00:00Z",
                "date": "2026-05-08", "bin_types": ["Food Waste"],
            })
        )
        client = BinNotifierClient("https://api.example", "tok")
        got = await client.get_next_collection("Home", bin_type="Food Waste")
        assert got["date"] == "2026-05-08"
        called = route.calls.last
        assert called.request.url.params["type"] == "Food Waste"
        assert called.request.headers["authorization"] == "Bearer tok"


@pytest.mark.asyncio
async def test_404_without_code_raises_not_found():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(404, json={"error": "not found"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(NotFound):
            await client.get_next_collection("Home")


@pytest.mark.asyncio
async def test_503_raises_no_data():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(503, json={"error": "no data", "code": "no_data"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(NoData):
            await client.get_next_collection("Home")


@pytest.mark.asyncio
async def test_other_errors_raise_api_error():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations").mock(return_value=httpx.Response(500, text="boom"))
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(ApiError):
            await client.list_locations()


@pytest.mark.asyncio
async def test_404_unknown_location_raises_unknown_location():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Hom/collections/next").mock(
            return_value=httpx.Response(404, json={"error": "no such location: Hom", "code": "unknown_location"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(UnknownLocation):
            await client.get_next_collection("Hom")


@pytest.mark.asyncio
async def test_404_no_match_raises_no_match():
    async with respx.mock(base_url="https://api.example") as mock:
        mock.get("/v1/locations/Home/collections/next").mock(
            return_value=httpx.Response(404, json={"error": "no matching collection", "code": "no_match"})
        )
        client = BinNotifierClient("https://api.example", "tok")
        with pytest.raises(NoMatch):
            await client.get_next_collection("Home")
