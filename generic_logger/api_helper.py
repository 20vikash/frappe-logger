import jwt
from jwt import PyJWKClient

JWKS_URL = "http://188.245.72.65:3000/api/signing-keys/keys"

jwk_client = PyJWKClient(JWKS_URL)

def verify_jwt(token_string: str):
    try:
        signing_key = jwk_client.get_signing_key_from_jwt(token_string)

        decoded = jwt.decode(
            token_string,
            signing_key.key,
            algorithms=["ES256"],
            options={
                "verify_aud": False,
            },
            leeway=300
        )

        return decoded

    except Exception as e:
        raise Exception(f"JWT verification failed: {str(e)}")
