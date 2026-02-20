app_name = "generic_logger"
app_title = "Generic Logger"
app_publisher = "Vikash"
app_description = "Logger"
app_email = "vikash@frappe.io"
app_license = "mit"

# Apps
# ------------------

required_apps = ["press_agent_manager"]

# Each item in the list will be shown as an app in the apps page
# add_to_apps_screen = [
# 	{
# 		"name": "generic_logger",
# 		"logo": "/assets/generic_logger/logo.png",
# 		"title": "Generic Logger",
# 		"route": "/generic_logger",
# 		"has_permission": "generic_logger.api.permission.has_app_permission"
# 	}
# ]

# Includes in <head>
# ------------------

# include js, css files in header of desk.html
# app_include_css = "/assets/generic_logger/css/generic_logger.css"
# app_include_js = "/assets/generic_logger/js/generic_logger.js"

# include js, css files in header of web template
# web_include_css = "/assets/generic_logger/css/generic_logger.css"
# web_include_js = "/assets/generic_logger/js/generic_logger.js"

# include custom scss in every website theme (without file extension ".scss")
# website_theme_scss = "generic_logger/public/scss/website"

# include js, css files in header of web form
# webform_include_js = {"doctype": "public/js/doctype.js"}
# webform_include_css = {"doctype": "public/css/doctype.css"}

# include js in page
# page_js = {"page" : "public/js/file.js"}

# include js in doctype views
# doctype_js = {"doctype" : "public/js/doctype.js"}
# doctype_list_js = {"doctype" : "public/js/doctype_list.js"}
# doctype_tree_js = {"doctype" : "public/js/doctype_tree.js"}
# doctype_calendar_js = {"doctype" : "public/js/doctype_calendar.js"}

# Svg Icons
# ------------------
# include app icons in desk
# app_include_icons = "generic_logger/public/icons.svg"

# Home Pages
# ----------

# application home page (will override Website Settings)
# home_page = "login"

# website user home page (by Role)
# role_home_page = {
# 	"Role": "home_page"
# }

# Generators
# ----------

# automatically create page for each record of this doctype
# website_generators = ["Web Page"]

# automatically load and sync documents of this doctype from downstream apps
# importable_doctypes = [doctype_1]

# Jinja
# ----------

# add methods and filters to jinja environment
# jinja = {
# 	"methods": "generic_logger.utils.jinja_methods",
# 	"filters": "generic_logger.utils.jinja_filters"
# }

# Installation
# ------------

# before_install = "generic_logger.install.before_install"
# after_install = "generic_logger.install.after_install"

# Uninstallation
# ------------

# before_uninstall = "generic_logger.uninstall.before_uninstall"
# after_uninstall = "generic_logger.uninstall.after_uninstall"

# Integration Setup
# ------------------
# To set up dependencies/integrations with other apps
# Name of the app being installed is passed as an argument

# before_app_install = "generic_logger.utils.before_app_install"
# after_app_install = "generic_logger.utils.after_app_install"

# Integration Cleanup
# -------------------
# To clean up dependencies/integrations with other apps
# Name of the app being uninstalled is passed as an argument

# before_app_uninstall = "generic_logger.utils.before_app_uninstall"
# after_app_uninstall = "generic_logger.utils.after_app_uninstall"

# Desk Notifications
# ------------------
# See frappe.core.notifications.get_notification_config

# notification_config = "generic_logger.notifications.get_notification_config"

# Permissions
# -----------
# Permissions evaluated in scripted ways

# permission_query_conditions = {
# 	"Event": "frappe.desk.doctype.event.event.get_permission_query_conditions",
# }
#
# has_permission = {
# 	"Event": "frappe.desk.doctype.event.event.has_permission",
# }

# Document Events
# ---------------
# Hook on document methods and events

# doc_events = {
# 	"*": {
# 		"on_update": "method",
# 		"on_cancel": "method",
# 		"on_trash": "method"
# 	}
# }

# Scheduled Tasks
# ---------------

# scheduler_events = {
# 	"all": [
# 		"generic_logger.tasks.all"
# 	],
# 	"daily": [
# 		"generic_logger.tasks.daily"
# 	],
# 	"hourly": [
# 		"generic_logger.tasks.hourly"
# 	],
# 	"weekly": [
# 		"generic_logger.tasks.weekly"
# 	],
# 	"monthly": [
# 		"generic_logger.tasks.monthly"
# 	],
# }

# Testing
# -------

# before_tests = "generic_logger.install.before_tests"

# Extend DocType Class
# ------------------------------
#
# Specify custom mixins to extend the standard doctype controller.
# extend_doctype_class = {
# 	"Task": "generic_logger.custom.task.CustomTaskMixin"
# }

# Overriding Methods
# ------------------------------
#
# override_whitelisted_methods = {
# 	"frappe.desk.doctype.event.event.get_events": "generic_logger.event.get_events"
# }
#
# each overriding function accepts a `data` argument;
# generated from the base implementation of the doctype dashboard,
# along with any modifications made in other Frappe apps
# override_doctype_dashboards = {
# 	"Task": "generic_logger.task.get_dashboard_data"
# }

# exempt linked doctypes from being automatically cancelled
#
# auto_cancel_exempted_doctypes = ["Auto Repeat"]

# Ignore links to specified DocTypes when deleting documents
# -----------------------------------------------------------

# ignore_links_on_delete = ["Communication", "ToDo"]

# Request Events
# ----------------
# before_request = ["generic_logger.utils.before_request"]
# after_request = ["generic_logger.utils.after_request"]

# Job Events
# ----------
# before_job = ["generic_logger.utils.before_job"]
# after_job = ["generic_logger.utils.after_job"]

# User Data Protection
# --------------------

# user_data_fields = [
# 	{
# 		"doctype": "{doctype_1}",
# 		"filter_by": "{filter_by}",
# 		"redact_fields": ["{field_1}", "{field_2}"],
# 		"partial": 1,
# 	},
# 	{
# 		"doctype": "{doctype_2}",
# 		"filter_by": "{filter_by}",
# 		"partial": 1,
# 	},
# 	{
# 		"doctype": "{doctype_3}",
# 		"strict": False,
# 	},
# 	{
# 		"doctype": "{doctype_4}"
# 	}
# ]

# Authentication and authorization
# --------------------------------

# auth_hooks = [
# 	"generic_logger.auth.validate"
# ]

# Automatically update python controller files with type annotations for this app.
# export_python_type_annotations = True

# default_log_clearing_doctypes = {
# 	"Logging DocType Name": 30  # days to retain logs
# }

# Translation
# ------------
# List of apps whose translatable strings should be excluded from this app's translations.
# ignore_translatable_strings_from = []

