import frappe
from generic_logger.api_helper import verify_jwt

@frappe.whitelist(allow_guest=True)
def get_log_user_meta(jwt_token: str):
    try:
        claims = verify_jwt(jwt_token)
        return {
            "valid": True,
            "claims": claims
        }
    except Exception as e:
        frappe.throw(str(e))
