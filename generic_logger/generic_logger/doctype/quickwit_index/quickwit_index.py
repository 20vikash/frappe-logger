# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

import json

import frappe
from frappe.model.document import Document
import requests


class QuickWitIndex(Document):
	def autoname(self):
		schema = frappe.parse_json(self.schema)
		self.name = schema.index_id

	@frappe.whitelist()
	def create_index(self):
		if not frappe.db.exists("QuickWit Index", self.name):
			frappe.throw("Save the document before creating index")

		if self.created:
			frappe.throw("Index already created")

		quickwit_server = frappe.get_doc("QuickWit Server", self.quickwit_server)
		virtual_machine = frappe.get_doc("Virtual Machine", quickwit_server.virtual_machine)

		host = virtual_machine.public_ip_address

		quickwit_index_url = "http://{}:7280/api/v1/indexes".format(host)

		headers = {
			"Content-Type": "application/json"
		}

		schema_dict = json.loads(self.schema)

		response = requests.post(
			quickwit_index_url,
			headers=headers,
			json=schema_dict
		)

		if response.status_code == 200:
			self.set("created", 1)
			self.save()

			frappe.msgprint("Successfully created index")
		else:
			frappe.throw("Failed to create index")

	@frappe.whitelist()
	def delete_index(self):
		quickwit_server = frappe.get_doc("QuickWit Server", self.quickwit_server)
		virtual_machine = frappe.get_doc("Virtual Machine", quickwit_server.virtual_machine)

		host = virtual_machine.public_ip_address

		if not self.created:
			frappe.throw("Index not yet created")
		
		index_id = self.name

		quickwit_index_url = "http://{}:7280/api/v1/indexes/{}/".format(host, index_id)

		response = requests.delete(
			quickwit_index_url
		)

		if response.status_code == 200:
			self.set("created", 0)
			self.save()
			frappe.msgprint("Successfully deleted index")
		else:
			frappe.throw("Failed to delete index")
