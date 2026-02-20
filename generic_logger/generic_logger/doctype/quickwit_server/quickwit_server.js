// Copyright (c) 2026, Vikash and contributors
// For license information, please see license.txt

frappe.ui.form.on("QuickWit Server", {
	refresh(frm) {
		[["Provision", "provision"]].forEach(([label, method]) => {
			frm.add_custom_button(
				label,
				() => {
					// Ask confirmation
					frappe.confirm(
						`Are you sure you want to ${label.toLowerCase()} this quickwit server?`,
						() => {
							frm.call(method).then(() => frm.refresh());
						},
					);
				},
				"Actions",
			);
		});
	},
});
