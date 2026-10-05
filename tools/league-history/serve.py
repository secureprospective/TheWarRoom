#!/usr/bin/env python3
"""Serve only PWA assets and the generated private index, never the raw archive."""
import argparse
import ipaddress
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote, urlsplit


class HistoryHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        if not self.host_allowed():
            self.send_error(403)
        elif self.allowed():
            super().do_GET()
        else:
            self.send_error(404)

    def do_HEAD(self):
        if not self.host_allowed():
            self.send_error(403)
        elif self.allowed():
            super().do_HEAD()
        else:
            self.send_error(404)

    def host_allowed(self):
        # Loopback binding alone does not prevent browser DNS-rebinding reads.
        try:
            host = urlsplit('//'+self.headers.get('Host', '')).hostname
            host = host.lower().rstrip('.') if host else ''
            bind = self.server.server_address[0]
            if host in {'localhost', '127.0.0.1', bind, self.server.server_name.lower().rstrip('.')}:
                return True
            return bind == '0.0.0.0' and ipaddress.ip_address(host).is_private
        except ValueError:
            return False

    def allowed(self):
        path = unquote(urlsplit(self.path).path)
        return path in {'/', '/index.html', '/style.css', '/app.js', '/model.mjs', '/behavior.mjs', '/behavior_views.mjs', '/history_views.mjs', '/sw.js',
                        '/manifest.webmanifest', '/icon.svg', '/icon-192.png', '/icon-512.png',
                        '/data/archive.json', '/data/revision.json'}

    def end_headers(self):
        self.send_header('Cache-Control', 'no-cache')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.send_header('Referrer-Policy', 'no-referrer')
        self.send_header('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
        super().end_headers()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--port', type=int, default=8765)
    parser.add_argument('--bind', default='127.0.0.1')
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    if not (root / 'data/archive.json').is_file():
        parser.error('Missing generated index. Run python3 build_archive.py first.')
    handler = partial(HistoryHandler, directory=str(root))
    server = ThreadingHTTPServer((args.bind, args.port), handler)
    print(f'Legacy NFL history: http://{args.bind}:{args.port}/', flush=True)
    print('localhost supports PWA installation. LAN HTTP is online-only; use HTTPS for other devices.', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.server_close()


if __name__ == '__main__':
    main()
