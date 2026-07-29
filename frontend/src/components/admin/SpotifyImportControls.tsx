import { useState, type FormEvent } from "react";
import { searchSpotifyArtists, searchSpotifyTracks } from "../../api/admin";
import { ApiRequestError } from "../../api/client";
import type { SpotifyArtistSearchItem, SpotifyTrackSearchItem } from "../../types/admin";

type SearchState<T> = {
  items: T[];
  loading: boolean;
  error: string;
};

const initialState = <T,>(): SearchState<T> => ({ items: [], loading: false, error: "" });

function errorMessage(error: unknown): string {
  if (error instanceof ApiRequestError) return error.response.message;
  return "Spotify検索に失敗しました";
}

export function SpotifyArtistImport({ onSelect }: { onSelect: (artist: SpotifyArtistSearchItem) => void }) {
  const [query, setQuery] = useState("");
  const [state, setState] = useState<SearchState<SpotifyArtistSearchItem>>(initialState);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = query.trim();
    if (!value) return;
    setState((current) => ({ ...current, loading: true, error: "" }));
    try {
      const result = await searchSpotifyArtists(value);
      setState({ items: result.items, loading: false, error: "" });
    } catch (error: unknown) {
      setState({ items: [], loading: false, error: errorMessage(error) });
    }
  }

  return (
    <section className="spotify-import" aria-label="Spotify Artist検索">
      <h2>SpotifyからArtistを検索</h2>
      <form className="spotify-import-search" onSubmit={(event) => void submit(event)}>
        <input aria-label="Spotify Artist検索キーワード" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Artist名" />
        <button type="submit" disabled={state.loading || query.trim() === ""}>{state.loading ? "検索中..." : "検索"}</button>
      </form>
      {state.error && <p className="spotify-import-error" role="alert">{state.error}</p>}
      <div className="spotify-import-results">
        {state.items.map((artist) => (
          <article key={artist.id} className="spotify-import-card">
            {artist.images[0]?.url ? <img src={artist.images[0].url} alt="" /> : <div className="spotify-import-placeholder">No image</div>}
            <div><strong>{artist.name}</strong><span>Popularity: {artist.popularity}</span><small>{artist.genres.join(", ") || "Genre情報なし"}</small></div>
            <button type="button" onClick={() => onSelect(artist)}>登録フォームへ</button>
          </article>
        ))}
      </div>
    </section>
  );
}

export function SpotifyTrackImport({ onSelect }: { onSelect: (track: SpotifyTrackSearchItem) => void }) {
  const [query, setQuery] = useState("");
  const [state, setState] = useState<SearchState<SpotifyTrackSearchItem>>(initialState);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = query.trim();
    if (!value) return;
    setState((current) => ({ ...current, loading: true, error: "" }));
    try {
      const result = await searchSpotifyTracks(value);
      setState({ items: result.items, loading: false, error: "" });
    } catch (error: unknown) {
      setState({ items: [], loading: false, error: errorMessage(error) });
    }
  }

  return (
    <section className="spotify-import" aria-label="Spotify Song検索">
      <h2>SpotifyからSongを検索</h2>
      <p>先にArtist管理で対象Artistを登録してください。</p>
      <form className="spotify-import-search" onSubmit={(event) => void submit(event)}>
        <input aria-label="Spotify Song検索キーワード" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="曲名またはArtist名" />
        <button type="submit" disabled={state.loading || query.trim() === ""}>{state.loading ? "検索中..." : "検索"}</button>
      </form>
      {state.error && <p className="spotify-import-error" role="alert">{state.error}</p>}
      <div className="spotify-import-results">
        {state.items.map((track) => (
          <article key={track.id} className="spotify-import-card">
            {track.album.images[0]?.url ? <img src={track.album.images[0].url} alt="" /> : <div className="spotify-import-placeholder">No image</div>}
            <div><strong>{track.name}</strong><span>{track.artists.map((artist) => artist.name).join(", ")}</span><small>{track.album.name}</small></div>
            <button type="button" onClick={() => onSelect(track)}>登録フォームへ</button>
          </article>
        ))}
      </div>
    </section>
  );
}
