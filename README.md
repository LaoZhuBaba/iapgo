# iapgo
#### A configuration wrapper for launching Google Identity Aware Proxy (IAP) tunnels

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
default:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  local_port: 1234
  remote_port: 1433
  remote_nic: nic0
  exec:
    - bash
    - "-c"
    - sqlcmd -S tcp:localhost,1234 -U fred -No -d mydb -Q 'SELECT * FROM mytable'
example:
  project_id: my-gcp-project
  zone: us-central1-a
  instance: my-jumpbox
  local_port: 1111
  remote_port: 1111
  remote_nic: nic0
  exec:
    - bash
    - "-c"
    - command and arguments
```
