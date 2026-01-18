# webhook.py
from http.server import BaseHTTPRequestHandler, HTTPServer
import json

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        alert = json.loads(post_data.decode('utf-8'))
        print("🚨 收到告警:", alert)
        self.send_response(200)
        self.end_headers()

if __name__ == "__main__":
    server = HTTPServer(('localhost', 8081), WebhookHandler)
    print("Webhook listening on http://localhost:8081")
    server.serve_forever()