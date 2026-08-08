from http.server import BaseHTTPRequestHandler, HTTPServer
import json
from urllib.parse import urlparse

ARTIST = {"id": "mock-artist-1", "name": "Mock Artist", "popularity": 50,
          "external_urls": {"spotify": "https://open.spotify.com/artist/mock-artist-1"},
          "images": [{"url": "https://example.test/artist.jpg"}]}
TRACK = {"id": "mock-track-1", "name": "Mock Song", "popularity": 50,
         "duration_ms": 180000, "explicit": False, "preview_url": None,
         "external_urls": {"spotify": "https://open.spotify.com/track/mock-track-1"},
         "artists": [ARTIST], "album": {"name": "Mock Album", "release_date": "2026-01-01",
         "images": [{"url": "https://example.test/song.jpg"}]}}

class Handler(BaseHTTPRequestHandler):
    def send_json(self, status, body):
        payload = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        path = urlparse(self.path).path
        if path == "/health":
            return self.send_json(200, {"status": "ok"})
        if path.endswith("/search"):
            return self.send_json(200, {"artists": {"items": [ARTIST], "total": 1}, "tracks": {"items": [TRACK], "total": 1}})
        if "/artists/" in path:
            return self.send_json(200, ARTIST)
        if "/tracks/" in path:
            return self.send_json(200, TRACK)
        return self.send_json(404, {"error": {"status": 404, "message": "not found"}})

    def do_POST(self):
        if self.path.endswith("/api/token"):
            return self.send_json(200, {"access_token": "mock-access-token", "token_type": "Bearer", "expires_in": 3600})
        return self.send_json(404, {"error": "not found"})

    def log_message(self, *_):
        return

HTTPServer(("0.0.0.0", 9090), Handler).serve_forever()
