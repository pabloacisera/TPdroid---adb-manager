#!/usr/bin/env python3
"""Mock license server for testing the TPDroid activation flow."""
import http.server
import json
import hmac
import hashlib
import sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8787
SECRET = "test-secret-for-mock-worker"

class MockWorkerHandler(http.server.BaseHTTPRequestHandler):
    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_POST(self):
        if self.path == "/activar":
            self.handle_activar()
        elif self.path == "/revalidar":
            self.handle_revalidar()
        else:
            self.send_error(404)

    def handle_activar(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))
        codigo = body.get("codigo", "")
        hw_id = body.get("hw_id", "")
        email = body.get("email", "")

        issued = "2026-06-29T12:00:00Z"
        payload = codigo + hw_id + issued
        sig = hmac.new(SECRET.encode(), payload.encode(), hashlib.sha256).hexdigest()

        lic = {"codigo": codigo, "hw_id": hw_id, "issued": issued, "hmac": sig}
        resp = {"success": True, "message": "Licencia activada correctamente", "lic": lic}

        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(json.dumps(resp).encode())

    def handle_revalidar(self):
        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))
        lic = body.get("lic", {})
        current_hw_id = body.get("current_hw_id", "")

        expected_payload = lic.get("codigo", "") + lic.get("hw_id", "") + lic.get("issued", "")
        expected_sig = hmac.new(SECRET.encode(), expected_payload.encode(), hashlib.sha256).hexdigest()

        if lic.get("hmac") != expected_sig:
            resp = {"error": "Firma de licencia invalida"}
            code = 401
        elif lic.get("hw_id") != current_hw_id:
            resp = {"error": "Esta licencia no corresponde a este equipo"}
            code = 403
        else:
            resp = {"success": True, "message": "Licencia valida", "hw_id": current_hw_id, "issued": lic.get("issued")}
            code = 200

        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        self.wfile.write(json.dumps(resp).encode())

    def log_message(self, format, *args):
        print(f"[mock-worker] {args[0]}")

server = http.server.HTTPServer(("127.0.0.1", PORT), MockWorkerHandler)
print(f"[mock-worker] Running on http://127.0.0.1:{PORT}")
server.serve_forever()
