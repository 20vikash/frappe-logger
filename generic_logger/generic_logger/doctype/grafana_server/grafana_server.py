# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

import frappe
from frappe.model.document import Document


class GrafanaServer(Document):
	def before_insert(self):
		if not self.virtual_machine:
			frappe.throw("Virtual Machine field is missing")

		vm = frappe.get_doc("Virtual Machine", self.virtual_machine)

		if not self.oauth:
			oauth_client = frappe.get_doc({
				"doctype": "OAuth Client",
				"app_name": "Grafana",
				"default_redirect_uri": "http://{}:3000/login/generic_oauth".format(vm.public_ip_address),
				"redirect_uris": "http://{}:3000/login/generic_oauth".format(vm.public_ip_address),
				"scopes": "openid profile user:email",
			})
			oauth_client.insert()

			self.oauth = oauth_client

	@frappe.whitelist()
	def provision(self):
		if not frappe.db.exists("Grafana Server", self.name):
			frappe.throw("Save the document before provisioning")

		quickwit_index = frappe.get_doc("QuickWit Index", self.quickwit_index)

		vm = frappe.get_doc("Virtual Machine", self.virtual_machine)
		grafana_host = vm.public_ip_address

		oAuth_client = frappe.get_doc("OAuth Client", self.oauth)

		client_id = oAuth_client.client_id
		client_secret = oAuth_client.client_secret
		admin_user = self.admin_user
		admin_password = self.admin_password

		variables = {
			"grafana_oauth_client_id": client_id,
			"grafana_oauth_client_secret": client_secret,
			"grafana_admin_user": admin_user,
			"grafana_admin_password": admin_password,
			"grafana_oauth_auth_url": "http://{}:8000/api/method/frappe.integrations.oauth2.authorize".format("188.245.72.102"),
			"grafana_oauth_token_url": "http://{}:8000/api/method/frappe.integrations.oauth2.get_token".format("188.245.72.102"),
			"grafana_oauth_api_url": "http://{}:8000/api/method/frappe.integrations.oauth2.openid_profile".format("188.245.72.102"),
			"grafana_domain": "{}:3000".format(grafana_host),
			"grafana_root_url": "http://{}:3000".format(grafana_host),
			"quickwit_url": "http://{}:8080/api/v1".format("188.245.72.65"),
			"quickwit_index": quickwit_index.name,
			"grafana_image": "docker.io/grafana/grafana-enterprise:latest",
			"grafana_port": 3000,
			"grafana_service_name": "grafana",
			"grafana_data_dir": "/var/lib/grafana",
			"grafana_config_dir": "/etc/grafana",
			"grafana_quadlet_dir": "/etc/containers/systemd",
		}

		play = vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/grafana.yml", run_in_background=True, variables=variables)

		frappe.msgprint(f"Created Ansible Play <a href='{play.get_url()}'>View Play</a>")
