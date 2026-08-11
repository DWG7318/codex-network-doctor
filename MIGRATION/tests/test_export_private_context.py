import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class ExportPrivateContextTests(unittest.TestCase):
    def test_redacts_secrets_and_stops_before_marker(self):
        repository_root = Path(__file__).resolve().parents[2]
        exporter = repository_root / "MIGRATION" / "export_private_context.py"
        self.assertTrue(exporter.is_file(), "export_private_context.py is missing")

        with tempfile.TemporaryDirectory(prefix="lcnd-session-export-") as temp:
            temp_root = Path(temp)
            source = temp_root / "source.jsonl"
            destination = temp_root / "sanitized.jsonl"
            records = [
                {
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": "assistant",
                        "content": [{"type": "output_text", "text": "unrelated LCSP details"}],
                    },
                },
                {
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": "user",
                        "content": [
                            {
                                "type": "input_image",
                                "image_url": "data:image/png;base64,LCSP",
                            }
                        ],
                    },
                },
                {
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": "assistant",
                        "encrypted_content": "opaque-ciphertext-that-is-not-portable",
                        "content": [
                            {
                                "type": "output_text",
                                "text": (
                                    "token=" + "gho_" + "abcdefghijklmnopqrstuvwxyz "
                                    "url=socks5://alice:" + "secret" + "@example.invalid:1080 "
                                    'config={"password":"cleartext"} '
                                    r"key=C:\Users\Owner\AppData\Local\LCapi\ssh\deploy-ed25519"
                                ),
                            }
                        ],
                    },
                },
                {
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": "user",
                        "content": [{"type": "input_text", "text": "MIGRATION-START"}],
                    },
                },
                {
                    "type": "response_item",
                    "payload": {
                        "type": "message",
                        "role": "assistant",
                        "content": [{"type": "output_text", "text": "LCSP must not be copied"}],
                    },
                },
            ]
            source.write_text("".join(json.dumps(item) + "\n" for item in records), encoding="utf-8")

            completed = subprocess.run(
                [
                    sys.executable,
                    str(exporter),
                    "--source",
                    str(source),
                    "--destination",
                    str(destination),
                    "--stop-before",
                    "MIGRATION-START",
                    "--exclude-text",
                    "LCSP",
                ],
                check=False,
                capture_output=True,
                text=True,
            )
            self.assertEqual(completed.returncode, 0, completed.stderr)
            output = destination.read_text(encoding="utf-8")
            self.assertIn("<GITHUB_TOKEN_REDACTED>", output)
            self.assertIn("socks5://<URL_USERINFO_REDACTED>@example.invalid:1080", output)
            self.assertIn('<SECRET_VALUE_REDACTED>', output)
            self.assertIn("<SSH_PRIVATE_KEY_PATH_REDACTED>", output)
            self.assertIn("<ENCRYPTED_CONTENT_REMOVED>", output)
            self.assertNotIn("opaque-ciphertext-that-is-not-portable", output)
            self.assertIn("data:image/png;base64,LCSP", output)
            self.assertNotIn("secret", output)
            self.assertNotIn("unrelated LCSP details", output)
            self.assertNotIn("MIGRATION-START", output)


if __name__ == "__main__":
    unittest.main()
