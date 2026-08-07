import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { searchSpotifyArtists, searchSpotifyTracks } from "../../api/admin";
import { ApiRequestError } from "../../api/client";
import { SpotifyArtistImport, SpotifyTrackImport } from "./SpotifyImportControls";

vi.mock("../../api/admin", () => ({
  searchSpotifyArtists: vi.fn(),
  searchSpotifyTracks: vi.fn(),
}));
const artistsMock = vi.mocked(searchSpotifyArtists);
const tracksMock = vi.mocked(searchSpotifyTracks);

beforeEach(() => {
  artistsMock.mockReset();
  tracksMock.mockReset();
});

describe("Spotify import controls", () => {
  it("searches artists and selects a result", async () => {
    const onSelect = vi.fn();
    artistsMock.mockResolvedValue({
      total: 1,
      items: [{ spotify_artist_id: "artist-1", name: "Artist", genres: ["rock"], image_url: "", spotify_url: "https://spotify/artist", popularity: 80 }],
    });
    const user = userEvent.setup();
    render(<SpotifyArtistImport onSelect={onSelect} />);

    const input = screen.getByLabelText("Spotify Artist検索キーワード");
    expect(screen.getByRole("button", { name: "検索" })).toBeDisabled();
    await user.type(input, " Artist ");
    await user.click(screen.getByRole("button", { name: "検索" }));

    expect(await screen.findByText("Popularity: 80")).toBeInTheDocument();
    expect(artistsMock).toHaveBeenCalledWith("Artist");
    await user.click(screen.getByRole("button", { name: "登録フォームへ" }));
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ spotify_artist_id: "artist-1" }));
  });

  it("shows Spotify API errors", async () => {
    artistsMock.mockRejectedValue(new ApiRequestError({ status: 502, code: "SPOTIFY_ERROR", message: "Spotify unavailable", request_id: "req-spotify" }));
    const user = userEvent.setup();
    render(<SpotifyArtistImport onSelect={vi.fn()} />);
    await user.type(screen.getByLabelText("Spotify Artist検索キーワード"), "Artist");
    await user.click(screen.getByRole("button", { name: "検索" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Spotify unavailable");
  });

  it("searches tracks and selects a result", async () => {
    const onSelect = vi.fn();
    tracksMock.mockResolvedValue({
      total: 1,
      items: [{
        spotify_track_id: "track-1",
        name: "Song",
        artists: [{ spotify_artist_id: "artist-1", name: "Artist" }],
        album_name: "Album",
        release_date: "2026-01-01",
        image_url: "",
        duration_ms: 200000,
        explicit: false,
        review_status: "APPROVED",
        popularity: 70,
        preview_url: "",
        spotify_url: "https://spotify/track",
      }],
    });
    const user = userEvent.setup();
    render(<SpotifyTrackImport onSelect={onSelect} />);

    await user.type(screen.getByLabelText("Spotify Song検索キーワード"), "Song");
    await user.click(screen.getByRole("button", { name: "検索" }));
    expect(await screen.findByText("Album")).toBeInTheDocument();
    expect(tracksMock).toHaveBeenCalledWith("Song");
    await user.click(screen.getByRole("button", { name: "登録フォームへ" }));
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ spotify_track_id: "track-1" }));
  });
});
