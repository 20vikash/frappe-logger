import frappe
import requests
import json

@frappe.whitelist()
def test_standalone(*args,**kwargs):
    """
    This function will be executed when the Provision Action Button will be clicked
    """
    # The data is transmitted via keyword argument
    doc_json = frappe.parse_json(kwargs.get('doc'))
    name = doc_json.get('name')

    if not name:
        frappe.throw("Name is required")
    
    doc = frappe.get_doc('QuickWit Server', name)
    doc.check_permission("write")

    access_token = doc.get_password("s3_access_token")
    secret_key = doc.get_password("s3_secret_key")
    print("access key", access_token)
    print("secret key", secret_key)

    endpoint_url = doc.endpoint_url
    region = doc.region

    vm = frappe.get_doc("Virtual Machine", doc.virtual_machine)

    variables = {
        "S3_REGION": region,
        "S3_ENDPOINT": endpoint_url,
        "ACCESS_TOKEN": access_token,
        "SECRET_KEY": secret_key,
        "S3_BUCKET": doc.s3_bucket,
        "quickwit_config_dir": "/etc/quickwit"
    }

    vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/bootstrap.yml", run_in_background=False, variables=variables)
    play = vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/quickwit.yml", run_in_background=True, variables=variables)

    frappe.msgprint(f"Created Ansible Play <a href='{play.get_url()}'>View Play</a>")

@frappe.whitelist()
def test_index_api(*args,**kwargs):
    """
    This function will be executed when the Create Action Button will be clicked
    """
    # The data is transmitted via keyword argument
    doc_json = frappe.parse_json(kwargs.get('doc'))
    name = doc_json.get('name')
    doc = frappe.get_doc('QuickWit Index', name)

    if doc.created:
        frappe.throw("Index already created")

    if doc.docstatus != 1:
        frappe.throw("Document is not submitted")

    quickwit_server = frappe.get_doc("QuickWit Server", doc.quickwit_server)
    virtual_machine = frappe.get_doc("Virtual Machine", quickwit_server.virtual_machine)

    host = virtual_machine.public_ip_address

    quickwit_index_url = "http://{}:7280/api/v1/indexes".format(host)

    headers = {
        "Content-Type": "application/json"
    }

    print(doc.schema)
    schema_dict = json.loads(doc.schema)

    response = requests.post(
        quickwit_index_url,
        headers=headers,
        json=schema_dict
    )

    if response.status_code == 200:
        doc.db_set("created", 1)

    print("Status:", response.status_code)
    print("Response:", response.text)

@frappe.whitelist()
def delete_index(*args, **kwargs):
    doc_json = frappe.parse_json(kwargs.get('doc'))
    name = doc_json.get('name')
    doc = frappe.get_doc('QuickWit Index', name)

    quickwit_server = frappe.get_doc("QuickWit Server", doc.quickwit_server)
    virtual_machine = frappe.get_doc("Virtual Machine", quickwit_server.virtual_machine)

    host = virtual_machine.public_ip_address

    if not doc.created:
        frappe.throw("Index not yet created")

    if doc.docstatus != 2:
        frappe.throw("Document is not yet cancelled")
    
    index_id = doc.name

    quickwit_index_url = "http://{}:7280/api/v1/indexes/{}/".format(host, index_id)

    response = requests.delete(
        quickwit_index_url
    )

    if response.status_code == 200:
        doc.db_set("created", 0)

    print("Status:", response.status_code)
    print("Response:", response.text)

@frappe.whitelist()
def provision_grafana(*args, **kwargs):
    doc_json = frappe.parse_json(kwargs.get('doc'))
    name = doc_json.get('name')
    doc = frappe.get_doc('Grafana Server', name)

    quickwit_index = frappe.get_doc("QuickWit Index", doc.quickwit_index)

    vm = frappe.get_doc("Virtual Machine", doc.virtual_machine)
    grafana_host = vm.public_ip_address

    oAuth = frappe.get_doc("OAuth", doc.oauth)
    oAuth_client = frappe.get_doc("OAuth Client", oAuth.oauth_client)

    client_id = oAuth_client.client_id
    client_secret = oAuth_client.client_secret
    admin_user = doc.admin_user
    admin_password = doc.admin_password

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
        "quickwit_index": quickwit_index.name
    }

    vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/bootstrap.yml", run_in_background=False, variables=variables)
    play = vm.run_ansible_play(app="generic_logger", playbook_path="ansible/playbooks/grafana.yml", run_in_background=True, variables=variables)

    frappe.msgprint(f"Created Ansible Play <a href='{play.get_url()}'>View Play</a>")
