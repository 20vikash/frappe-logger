import frappe
from generic_logger.api_helper import verify_jwt

@frappe.whitelist(allow_guest=True)
def get_log_user_meta(jwt_token: str, jwks_url: str):
    try:
        claims = verify_jwt(jwt_token, jwks_url)

        email = claims.get("email")
        log_user = frappe.get_doc("Log User", email)

        return {
            "log_user": log_user.as_dict()
        }
    except Exception as e:
        frappe.throw(str(e))
