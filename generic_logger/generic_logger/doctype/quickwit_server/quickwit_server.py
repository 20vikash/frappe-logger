# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

# import frappe
from frappe.model.document import Document

class QuickWitServer(Document):
    def get_quickwit_server(self):
            self.status = "Ready"
            self.save()

            return "Hello world"
