import { beforeEach, describe, expect, it, vi } from "vitest";
import { request } from "./client";
import {
  changeUserRole,
  deleteArtist,
  deleteSong,
  deleteTag,
  getArtist,
  getAuditLog,
  getMonitoringAlerts,
  getMonitoringMetrics,
  getSong,
  getSystemLog,
  getTag,
  getUser,
  listArtists,
  listAuditLogs,
  listSongs,
  listSystemLogs,
  listTagOptions,
  listTags,
  listUsers,
  saveArtist,
  saveSong,
  saveTag,
  searchSpotifyArtists,
  searchSpotifyTracks,
  setUserStatus,
} from "./admin";

vi.mock("./client", () => ({ request: vi.fn() }));
const requestMock = vi.mocked(request);

beforeEach(() => {
  requestMock.mockReset();
  requestMock.mockResolvedValue({} as never);
});

describe("admin API", () => {
  it("builds list queries and omits empty values", async () => {
    await listArtists("rock", 2, { tag_id: "tag-1", explicit: undefined });
    await listSongs("", 3, { artist_id: "artist-1", explicit: false });
    await listTags("mood", 4, { category: "MOOD" });
    await listUsers("user@example.com", 5, { status: "active" });
    await listAuditLogs("login", 6, { action: "LOGIN" });
    await listSystemLogs("error", 7, { level: "ERROR" });

    expect(requestMock).toHaveBeenNthCalledWith(
      1,
      "/api/admin/artists?page=2&limit=20&keyword=rock&tag_id=tag-1",
      { method: "GET" },
    );
    expect(requestMock).toHaveBeenNthCalledWith(
      2,
      "/api/admin/songs?page=3&limit=20&artist_id=artist-1&explicit=false",
      { method: "GET" },
    );
    expect(requestMock).toHaveBeenNthCalledWith(
      3,
      "/api/admin/tags?page=4&limit=20&keyword=mood&category=MOOD",
      { method: "GET" },
    );
  });

  it("covers artist CRUD", async () => {
    await getArtist("artist-1");
    await saveArtist({ name: "Artist" });
    await saveArtist({ name: "Updated" }, "artist-1");
    await deleteArtist("artist-1");

    expect(requestMock).toHaveBeenNthCalledWith(1, "/api/admin/artists/artist-1", { method: "GET" });
    expect(requestMock).toHaveBeenNthCalledWith(2, "/api/admin/artists", {
      method: "POST",
      body: JSON.stringify({ name: "Artist" }),
    });
    expect(requestMock).toHaveBeenNthCalledWith(3, "/api/admin/artists/artist-1", {
      method: "PATCH",
      body: JSON.stringify({ name: "Updated" }),
    });
    expect(requestMock).toHaveBeenNthCalledWith(4, "/api/admin/artists/artist-1", { method: "DELETE" });
  });

  it("covers song and tag CRUD", async () => {
    await getSong("song-1");
    await saveSong({ name: "Song" });
    await saveSong({ name: "Updated" }, "song-1");
    await deleteSong("song-1");
    await listTagOptions();
    await getTag("tag-1");
    await saveTag({ name: "Mood" });
    await saveTag({ name: "Mood 2" }, "tag-1");
    await deleteTag("tag-1");

    expect(requestMock).toHaveBeenCalledWith("/api/admin/tags/options", { method: "GET" });
    expect(requestMock).toHaveBeenCalledWith("/api/admin/songs/song-1", { method: "DELETE" });
    expect(requestMock).toHaveBeenCalledWith("/api/admin/tags/tag-1", { method: "DELETE" });
  });

  it("covers system user operations and log detail APIs", async () => {
    await getUser("user-1");
    await setUserStatus("user-1", "suspended", "security review");
    await changeUserRole("user-1", "content_admin", "promotion");
    await getAuditLog("audit-1");
    await getSystemLog("log-1");

    expect(requestMock).toHaveBeenCalledWith("/api/system/users/user-1/status", {
      method: "PATCH",
      body: JSON.stringify({ status: "suspended", reason: "security review" }),
    });
    expect(requestMock).toHaveBeenCalledWith("/api/system/users/user-1/role", {
      method: "PATCH",
      body: JSON.stringify({ role: "content_admin", reason: "promotion" }),
    });
  });

  it("covers Spotify search and monitoring APIs", async () => {
    await searchSpotifyArtists("A B", 5);
    await searchSpotifyTracks("Song Name", 8);
    await getMonitoringMetrics();
    await getMonitoringAlerts();

    expect(requestMock).toHaveBeenCalledWith("/api/admin/spotify/artists?q=A+B&limit=5", { method: "GET" });
    expect(requestMock).toHaveBeenCalledWith("/api/admin/spotify/tracks?q=Song+Name&limit=8", { method: "GET" });
    expect(requestMock).toHaveBeenCalledWith("/api/system/metrics", { method: "GET" });
    expect(requestMock).toHaveBeenCalledWith("/api/system/alerts", { method: "GET" });
  });
});
