# iapgo
## This is a Work In Progress

### A configuration wrapper for launching Google Identity Aware Proxy (IAP) tunnels, either with or without a secondary SSH tunnel

### Example Usage for Cloud SQL DB Management:
![Alt Text](https://github.com/LaoZhuBaba/iapgo/blob/v2/iapgo.drawio.png)

Use this tool to create either a simple IAP tunnel to a port on a jump
box that is reachable via SSH or create a secondary SSH tunnel within the
IAP tunnel to reach an host that is reachable from the jump box.

This tool only automates the setup of tunnels that would otherwise be complex
to configure and maintain.  It is not designed to bypass any security.  IAP
tunnelling will only be possible if your firewall allows it, but it considered
more secure than allowing access from random public IP addresses.

This tool assumes that you have installed the Google Cloud SDK and have
logged in with sufficient privileges to connect via Identity Aware Proxy.

N.B.  You must set your Application Default Credentials, either with:
```
gcloud auth login [ACCOUNT] --update-adc
```
or
```
gcloud auth application-default login
```
See: https://cloud.google.com/docs/authentication/application-default-credentials

If you use the SSH tunnelling option then your account must have permission log into
the jump host via SSH.  Generally it is recommended to use Google *oslogin* for this.
See: https://cloud.google.com/compute/docs/oslogin.  If you use *oslogin* then *iapgo*
attempt to work out your SSH account name and setup is generally easier.

You will need to know the name the target GCE instance, what project it
is in, and the zone to which it is deployed.

You could find this information in Cloud Console or by running:

```
gcloud compute instances list [--project=my_project]
```
You  also need to ensure that the GCE instance is listening on the defined
remote_port and typically you will need a VPC firewall rule to allow
access to this TCP port from Google's IAP proxy server CIDR which is
35.235.240.0/20

Unless your target GCE instance has multiple network interfaces *remote_nic*
should always be set to *nic0*.

The *exec* command is optional but is useful if you want to run a particular
program when the tunnel starts.  For Linux/MacOS it is typically easiest
to run *bash -c* followed by a command that can include multiple words.  But
you could run a command directly without *bash -c*, but then you would
need to deal with any spaces yourself.  If you find this confusing then it
is similar to Dockerfile, so read up on that.  I presume that on Windows
*cmd /c* would work in a similar way.

An environment variable called *$IAP_LISTEN_PORT* is automatically set for
any command run by the *exec* statement.  The value of this variable will
be the port that the tunnel is listening on, regardless of whether the tunnel
is simple IAP or SSH with IAP, and regardless of whether the local_port
has been explicitly set by *local_port* or is an ephemeral port.
```
Usage:
iapgo [-c config_section] [-f config_file_name] [-v]

-c string
    select a non-default configuration file section (default "default")
-f string
    select a non-default configuration file (default "iapgo.yaml")
-h  print a usage message
-v  print debugging messages
```

Example configuration file:
```
# default will be used if no config section is specified
default:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  # If local_port is not set then an ephemeral port will be allocated and made available as $IAP_LISTEN_PORT
  # local_port: 1234
  remote_port: 80
  remote_nic: nic0
  exec:
    - bash
    - "-c"
    - curl http://localhost:$IAP_LISTEN_PORT
example:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  remote_port: 80 # When ssh_tunnel is used then this is the port on the tunnel_to hosts
  remote_nic: nic0
  ssh_tunnel:
    tunnel_to: 1.2.3.4 # This is a host that is reachable from my-jumpbox
    # If account_name is not set then an attempt will be made to get value from os-login
    # account_name: my_ssh_login
    # By default ~/.ssh/google_compute_engine will be used.
    # private_key_file: /home/fred/.ssh/google_compute_engine
  exec:
    - bash
    - "-c"
    # curl will reach ssh_tunnel.tunnel_to host on remote_port
    - curl http://localhost:$IAP_LISTEN_PORT
```
