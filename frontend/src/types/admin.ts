export interface PageResponse<T> { items: T[]; page: number; limit: number; total: number; total_pages: number }
export interface Artist { id: string; spotify_artist_id: string; name: string; image_url: string; spotify_url: string; popularity: number; created_at: string; updated_at: string }
export interface Song { id: string; artist_id: string; spotify_track_id: string; name: string; album_name: string; image_url: string; spotify_url: string; preview_url: string; duration_ms: number; popularity: number; release_date: string; explicit: boolean; created_at: string; updated_at: string }
export interface Tag { id: string; name: string; slug: string; category: string; description: string; created_at: string; updated_at: string }
export interface AdminUser { id: string; name: string; email: string; role: "user"|"content_admin"|"system_admin"; status: "active"|"suspended"; created_at: string; updated_at: string }
export interface AuditLog { id: string; actor_user_id: string; action: string; target_type: string; target_id: string; summary: string; ip_address: string; user_agent: string; created_at: string }
export interface SystemLog { id: string; level: string; category: string; message: string; request_id: string; user_id?: string; created_at: string }
