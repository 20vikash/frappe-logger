import ansible_runner

def run_playbook(**kwargs):
    # Run bootstrap playbook to install podman and set up config dir
    ansible_runner.run(
        playbook=kwargs["bootstrap_path"],
        inventory=kwargs["inventory_path"],
        extravars=kwargs["variables"],
        cmdline=f"--user=root",
	)

    # Run quickwit playbook to provision the quickwit server
    ansible_runner.run(
        playbook=kwargs["quickwit_path"],
        inventory=kwargs["inventory_path"],
        extravars=kwargs["variables"],
        cmdline=f"--user=root",
    )
