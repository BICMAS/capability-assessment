#!/usr/bin/env python3
"""Create the local uploads bucket. Existing buckets are left in place."""

import datetime
import hashlib
import hmac
import os
import sys
import urllib.error
import urllib.request
from urllib.parse import urlparse


def sign(key: bytes, message: str) -> bytes:
    return hmac.new(key, message.encode("utf-8"), hashlib.sha256).digest()


def main() -> int:
    access_key = os.environ["S3_ACCESS_KEY"]
    secret_key = os.environ["S3_SECRET_KEY"]
    bucket = os.environ["S3_BUCKET"]
    endpoint = os.environ.get("S3_ENDPOINT", "http://localhost:9000").rstrip("/")
    parsed = urlparse(endpoint)
    host = parsed.netloc
    now = datetime.datetime.now(datetime.timezone.utc)
    amz_date = now.strftime("%Y%m%dT%H%M%SZ")
    date_stamp = now.strftime("%Y%m%d")
    payload_hash = hashlib.sha256(b"").hexdigest()
    canonical_uri = f"/{bucket}"
    canonical_headers = f"host:{host}\nx-amz-content-sha256:{payload_hash}\nx-amz-date:{amz_date}\n"
    signed_headers = "host;x-amz-content-sha256;x-amz-date"
    canonical_request = "\n".join(
        ["PUT", canonical_uri, "", canonical_headers, signed_headers, payload_hash]
    )
    credential_scope = f"{date_stamp}/us-east-1/s3/aws4_request"
    string_to_sign = "\n".join(
        [
            "AWS4-HMAC-SHA256",
            amz_date,
            credential_scope,
            hashlib.sha256(canonical_request.encode("utf-8")).hexdigest(),
        ]
    )
    signing_key = sign(("AWS4" + secret_key).encode("utf-8"), date_stamp)
    signing_key = sign(signing_key, "us-east-1")
    signing_key = sign(signing_key, "s3")
    signing_key = sign(signing_key, "aws4_request")
    signature = hmac.new(signing_key, string_to_sign.encode("utf-8"), hashlib.sha256).hexdigest()
    authorization = (
        "AWS4-HMAC-SHA256 "
        f"Credential={access_key}/{credential_scope}, "
        f"SignedHeaders={signed_headers}, "
        f"Signature={signature}"
    )
    request = urllib.request.Request(
        f"{endpoint}{canonical_uri}",
        method="PUT",
        headers={
            "Host": host,
            "x-amz-content-sha256": payload_hash,
            "x-amz-date": amz_date,
            "Authorization": authorization,
        },
    )
    try:
        with urllib.request.urlopen(request) as response:
            print(f"created bucket {bucket} ({response.status})")
    except urllib.error.HTTPError as err:
        body = err.read().decode("utf-8", errors="replace")
        if err.code == 409 and "BucketAlreadyOwnedByYou" in body:
            print(f"bucket {bucket} already exists")
            return 0
        print(body, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
