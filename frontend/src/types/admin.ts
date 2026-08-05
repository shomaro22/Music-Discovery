export interface PageResponse<T> { items: T[]; page: number; limit: number; total: number; total_pages: number }
export interface TagRef { id: string; name: string; slug: string; category: string }
export interface Artist { id: string; spotify_artist_id: string; name: string; image_url: string; spotify_url: string; popularity: number; tags?: TagRef[]; song_count?: number; user_weight_count?: number; created_at: string; updated_at: string }
export interface SongTagRef extends TagRef { weight: number }
export type ContentReviewStatus = "PENDING" | "AUTO_APPROVED" | "APPROVED" | "REJECTED";
export interface Song { id: string; artist_id: string; spotify_track_id: string; name: string; album_name: string; image_url: string; spotify_url: string; preview_url: string; duration_ms: number; popularity: number; release_date: string; explicit: boolean; review_status: ContentReviewStatus; tags?: SongTagRef[]; artist?: Artist; recommendation_count?: number; feedback_count?: number; like_count?: number; maybe_count?: number; skip_count?: number; created_at: string; updated_at: string }
export interface Tag { id: string; name: string; slug: string; category: string; description: string; artist_count?: number; song_count?: number; user_weight_count?: number; created_at: string; updated_at: string }
export interface AdminUser { id: string; name: string; email: string; role: "user"|"content_admin"|"system_admin"; status: "active"|"suspended"; suspended_at?: string; suspended_reason?: string; suspended_by?: string; deleted_at?: string; created_at: string; updated_at: string; profile_status?: string; recommendation_count?: number; feedback_count?: number; last_login_at?: string }
export interface AuditLog { id: string; actor_user_id: string; actor_role: string; actor_name: string; actor_email: string; request_id: string; action: string; target_type: string; target_id: string; reason: string; before?: unknown; after?: unknown; summary: string; ip_address: string; user_agent: string; created_at: string }
export interface SystemLog { id: string; level: string; category: string; event: string; error_code: string; message: string; request_id: string; user_id?: string; method: string; path: string; status_code: number; external_service?: string; duration_ms?: number; error_detail?: string; ip_address?: string; user_agent?: string; metadata?: unknown; stack_trace?: string; created_at: string }

export interface SpotifyArtistSearchItem {
  spotify_artist_id: string;
  name: string;
  genres: string[];
  image_url: string;
  spotify_url: string;
  popularity: number;
}
export interface SpotifyTrackArtist {
  spotify_artist_id: string;
  name: string;
}
export interface SpotifyTrackSearchItem {
  spotify_track_id: string;
  name: string;
  artists: SpotifyTrackArtist[];
  album_name: string;
  release_date: string;
  image_url: string;
  duration_ms: number;
  explicit: boolean; review_status: ContentReviewStatus;
  popularity: number;
  preview_url: string;
  spotify_url: string;
}
export interface SpotifySearchResponse<T> { items: T[]; total: number }
