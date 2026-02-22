# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

import frappe
from frappe.model.document import Document


class LogUser(Document):
	def autoname(self):
		user = frappe.get_doc("User", self.user)
		self.name = user.email
