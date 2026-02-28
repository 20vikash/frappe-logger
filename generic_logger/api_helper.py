import jwt
from jwt import PyJWKClient

def verify_jwt(token_string: str, jwks_url: str):
    try:
        jwk_client = PyJWKClient(jwks_url)
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
