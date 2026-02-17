# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

# import frappe
from frappe.model.document import Document


class QuickWitIndex(Document):
	def get_quickwit_index(self):
		return "Hello world"
