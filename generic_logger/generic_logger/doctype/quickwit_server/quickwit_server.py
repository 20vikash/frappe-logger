# Copyright (c) 2026, Vikash and contributors
# For license information, please see license.txt

import frappe
from frappe.model.document import Document

class QuickWitServer(Document):
    @frappe.whitelist()
    def provision(self):
        if not frappe.db.exists("QuickWit Server", self.name):
            frappe.throw("Save the document before provisioning")

        access_token = self.get_password("s3_access_token")
        secret_key = self.get_password("s3_secret_key")

        api_token = self.get_password("api_token")
        api_secret = self.get_password("api_secret")

        endpoint_url = self.endpoint_url
        region = self.region

        vm = frappe.get_doc("Virtual Machine", self.virtual_machine)

        variables = {
            "S3_REGION": region,
            "S3_ENDPOINT": endpoint_url,
            "ACCESS_TOKEN": access_token,
            "SECRET_KEY": secret_key,
            "S3_BUCKET": self.s3_bucket,
            "quickwit_config_dir": "/etc/quickwit",
            "quickwit_service_name": "quickwit",
            "quickwit_image": "ghcr.io/20vikash/frappe-logger:latest",
            "quickwit_port": 7280,
            "quickwit_data_dir": "/var/lib/quickwit",
            "quickwit_quadlet_dir": "/etc/containers/systemd",
            "api_token": api_token,
            "api_secret": api_secret
        }

        play = vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/quickwit.yml", run_in_background=True, variables=variables)

        frappe.msgprint(f"Created Ansible Play <a href='{play.get_url()}'>View Play</a>")
