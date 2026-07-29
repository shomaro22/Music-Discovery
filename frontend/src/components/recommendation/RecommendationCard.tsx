import type { RecommendationDetail } from "../../types/recommendation";
const LABELS = { YOUR_MATCH: "Your Match", NEW_FAVORITE: "New Favorite", TRY_SOMETHING_NEW: "Try Something New" } as const;
interface RecommendationCardProps { detail: RecommendationDetail; }
export function RecommendationCard({ detail }: RecommendationCardProps) {
  return <article className="recommendation-card">
    <p className="recommendation-type">{LABELS[detail.recommendation_type]}</p>
    {detail.artwork_url ? <img src={detail.artwork_url} alt={`${detail.song_name} artwork`} /> : <div className="recommendation-artwork-placeholder" aria-label="artworkなし">♪</div>}
    <div className="recommendation-card-body"><h2>{detail.song_name}</h2><p className="recommendation-artist">{detail.artist_name}</p><p>{detail.reason}</p>
      <a className="spotify-button" href={detail.spotify_url} target="_blank" rel="noreferrer">Spotifyで聴く</a>
      <fieldset className="feedback-actions"><legend>この推薦はどうでしたか？</legend><button type="button" disabled>LIKE</button><button type="button" disabled>MAYBE</button><button type="button" disabled>SKIP</button></fieldset>
    </div>
  </article>;
}
