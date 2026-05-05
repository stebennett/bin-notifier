import pytest

from bin_notifier_mcp import server


class FakeClient:
    def __init__(self):
        self.locations = [{"label": "Home", "postcode": "RG12"}]
        self.next_response = {
            "location": "Home", "scraped_at": "2026-05-05T18:00:00Z",
            "date": "2026-05-07", "bin_types": ["General Waste"],
        }

    async def list_locations(self):
        return self.locations

    async def get_next_collection(self, label, *, bin_type=None, from_date=None):
        if bin_type == "Garden":
            from bin_notifier_mcp.client import NotFound
            raise NotFound("no matching collection")
        return self.next_response


@pytest.fixture
def env(monkeypatch, tmp_path):
    monkeypatch.setenv("BN_API_BASE_URL", "https://api.example")
    monkeypatch.setenv("BN_API_TOKEN", "tok")
    monkeypatch.setenv("BN_DEFAULT_LOCATION", "Home")
    return None


@pytest.mark.asyncio
async def test_list_locations_tool(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.list_locations_tool()
    assert out == [{"label": "Home", "postcode": "RG12"}]


@pytest.mark.asyncio
async def test_get_next_collection_uses_default_location(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_tool()
    assert out["location"] == "Home"
    assert out["date"] == "2026-05-07"
    assert out["bin_types"] == ["General Waste"]
    assert "days_until" in out


@pytest.mark.asyncio
async def test_get_next_collection_of_type_returns_message_when_missing(env, monkeypatch):
    fake = FakeClient()
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_of_type_tool("Garden")
    assert "no" in out["message"].lower()


@pytest.mark.asyncio
async def test_missing_default_location_with_multiple_locations_errors(monkeypatch):
    monkeypatch.setenv("BN_API_BASE_URL", "https://api.example")
    monkeypatch.setenv("BN_API_TOKEN", "tok")
    monkeypatch.delenv("BN_DEFAULT_LOCATION", raising=False)

    fake = FakeClient()
    fake.locations = [{"label": "Home", "postcode": "RG12"}, {"label": "Office", "postcode": "RG40"}]
    monkeypatch.setattr(server, "_make_client", lambda: fake)
    out = await server.get_next_collection_tool()
    assert "location" in out["error"].lower()
    assert "Home" in out["error"] and "Office" in out["error"]
