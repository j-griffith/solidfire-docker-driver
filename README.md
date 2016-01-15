SolidFire Plugin for Docker Volumes
======================================
Plugin for SolidFire Docker integration

## Description
This plugin provides the ability to use SolidFire Storage Clusters as backend
devices in a Docker environment.

## Architecture
The SolidFire plugin is made of multiple packages:
- The SolidFire Docker Service Daemon
- The SolidFire API (sfapi) modules providing our Go bindings
- The SolidFire admin CLI (sfcli) providing a CLI tool for administrative tasks

The service and the components are designed to be stateless with the exception
of the config file.  That means that **you MUST use the EXACT SAME config file on
each Docker node** if you want to be able to move volumes with their data to new
Containers that you migrate from one Docker host to another.

  * SolidFire Docker Service Daemon
  Standard Docker plugin model, we've moved away from using the Gorilla libs in
  favor of leveraging the excellent go-plugins-helpers packages instead.  More
  information can be found on it's github page here
  [go-plugins-helpers/volume]: https://github.com/docker/go-plugins-helpers/volume

  The Daemon simply accepts json requests that are routed from Docker to the
  SolidFire Driver which then routes the request to the appropriate SolidFire
  API call via a mux handler.

  * SolidFire API
  These are the GoLang bindings to issue commands to the SolidFire Cluster.
  Any request to the SolidFire cluster is made via a json-rpc request through
  this library.

  * SolidFire CLI
  Docker's API for volumes is currently pretty simple (that's a great thing),
  but sometimes there is a need for some Admin tasks and it's not always
  convenient to jump over to a web-browser or other tool.  For that reason we
  also include a basic CLI with a number of common features/tasks that are
  implemented.  For more information run the help command to see what commands
  are available and what their syntax is.


## Installation

Use the standard golang install process:
  ```
  go get -u github.com/solidfire/solidfire-docker-driver
  ```
The SolidFire plugin is made of multiple packages:
- The SolidFire Docker Service Daemon
- The SolidFire API (sfapi) modules providing our Go bindings
- The SolidFire admin CLI (sfcli) providing a CLI tool for administrative tasks
  including starting the daemon/service.

* Each of the following packages needs to be installed on EVERY Docker Node:

- Open-iSCSI
  * Ubuntu<br>
  ```
  sudo apt-get install open-iscsi
  ```

  * Redhat variants<br>
  ```
  sud yum in stall iscsi-initiator-utils
  ```

Now simply start the SolidFire daemon:
  ```
  solidfire-docker-driver daemon start
  ```

This package is tested on and requires Docker version >= 1.9.1

## Configuration
During startup of the SolidFire Docker service, the plugin obtains it's setting
information from a provided config file.  The config file can be specified via
the command line on startup, and also by default the service will attempt to
find a config at the default location:
  ```
  /var/lib/solidfire/solidfire.json
  ```

The SolidFire config file is a minimal config file that includes basic
information abou the SolidFire cluster to use, and includes specification of a
tenant account to create and (or) use on the SolidFire cluster.  It also
includes directives to specify where volumes should be mounted on the Docker
host.  Here's an example solidfire.json config file:

  ```
  {
    "Endpoint": "https://admin:admin@192.168.160.3/json-rpc/7.0",
    "SVIP": "10.10.64.3:3260",
    "TenantName": "docker",
    "DefaultVolSize": 1,
    "MountPoint": "/var/lib/solidfire/mount",
    "Types": [{"Type": "Bronze", "Qos": {"minIOPS": 1000, "maxIOPS": 2000, "burstIOPS": 4000}},
              {"Type": "Silver", "Qos": {"minIOPS": 4000, "maxIOPS": 6000, "burstIOPS": 8000}},
              {"Type": "Gold", "Qos": {"minIOPS": 6000, "maxIOPS": 8000, "burstIOPS": 10000}}]

  }
  ```

Note the format of the endpoint is https://\<login\>:\<password\>@\<mvip\>/json-rpc/\<element-version\>

Types are used to set desired QoS of Volumes via docker volume create opts.
You're free to create as many types as you wish.

Please note that at this time the Docker plugin for SolidFire ONLY supports
iSCSI and utilizes CHAP security for iSCSI connections.  FC support may or may
not be added in the future.

Licensing
---------
Copyright [2015] [SolidFire Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Support
-------
Please file bugs and issues at the Github issues page. For Docker specific questions/issues contact the Docker team. The code and documentation in this module are released with no warranties or SLAs and are intended to be supported via the Open Source community.
