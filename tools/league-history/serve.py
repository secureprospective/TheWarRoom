#!/usr/bin/env python3
"""Serve the League Behavior Lab on this machine only. Never serves the raw archive."""
import argparse
import gzip
import ipaddress
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parent
APP = ROOT / 'app'
DATA = ROOT / 'data'
TYPES = {'.js': 'text/javascript', '.css': 'text/css', '.html': 'text/html; charset=utf-8', '.json': 'application/json',
         '.svg': 'image/svg+xml', '.png': 'image/png', '.webmanifest': 'application/manifest+json'}


class LabHandler(SimpleHTTPRequestHandler):
    def host_allowed(self):
        # Loopback binding alone does not stop a browser DNS-rebinding read; check the Host header too.
        try:
            host = (urlsplit('//' + self.headers.get('Host', '')).hostname or '').lower().rstrip('.')
            bind = self.server.server_address[0]
            if host in {'localhost', '127.0.0.1', bind}:
                return True
            return bind == '0.0.0.0' and ipaddress.ip_address(host).is_private
        except ValueError:
            return False

    def resolve(self):
        path = unquote(urlsplit(self.path).path)
        if path in ('/', ''):
            path = '/index.html'
        if path in ('/data/lab.json', '/data/lab-revision.json'):
            return DATA / path.rsplit('/', 1)[1]
        target = (APP / path.lstrip('/')).resolve()
        if APP not in target.parents or not target.is_file() or target.suffix not in TYPES:
            return None
        return target

    def do_GET(self):
        self.respond(body=True)

    def do_HEAD(self):
        self.respond(body=False)

    def respond(self, body):
        if not self.host_allowed():
            return self.send_error(403)
        target = self.resolve()
        if target is None or not target.is_file():
            return self.send_error(404)
        content = target.read_bytes()
        gz = target.with_suffix(target.suffix + '.gz')
        encoded = False
        if 'gzip' in self.headers.get('Accept-Encoding', '') and (gz.is_file() and gz.stat().st_mtime >= target.stat().st_mtime):
            content, encoded = gz.read_bytes(), True
        elif 'gzip' in self.headers.get('Accept-Encoding', '') and target.suffix in ('.js', '.css', '.html', '.json') and len(content) > 2048:
            content, encoded = gzip.compress(content, 6), True
        self.send_response(200)
        self.send_header('Content-Type', TYPES[target.suffix])
        self.send_header('Content-Length', str(len(content)))
        if encoded:
            self.send_header('Content-Encoding', 'gzip')
        self.send_header('Vary', 'Accept-Encoding')
        self.end_headers()
        if body:
            self.wfile.write(content)

    def end_headers(self):
        self.send_header('Cache-Control', 'no-cache')
        self.send_header('X-Content-Type-Options', 'nosniff')
        self.send_header('Referrer-Policy', 'no-referrer')
        self.send_header('Content-Security-Policy', "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; "
                         "connect-src 'self'; worker-src 'self'; manifest-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
        super().end_headers()

    def log_message(self, fmt, *args):
        pass


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--port', type=int, default=8765)
    parser.add_argument('--bind', default='127.0.0.1')
    args = parser.parse_args()
    if not (DATA / 'lab.json').is_file():
        parser.error('The league data is missing. Run: python3 compile/build_lab.py')
    server = ThreadingHTTPServer((args.bind, args.port), partial(LabHandler, directory=str(APP)))
    print(f'Legacy NFL Behavior Lab: http://localhost:{args.port}/', flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        server.server_close()


if __name__ == '__main__':
    main()
