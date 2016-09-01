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
  [go-plugins-helpers/volume](https://github.com/docker/go-plugins-helpers/volume)

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


## Prerequisites
Linux OS system with Open-ISCSI tools.  Currently, the cli calls and most
of the API calls will work fine on Linux, or OSX, however all the attachment
code depends on iSCSI.  This driver does NOT support FibreChannel.

This package has been developed and tested using Docker version 1.9.1 and
Golang version 1.5.3 but should work fine with earlier versions. Typically
I don't use the distros packages but rather pull the latest versions of Go and
Docker from their respective websites.

The prerequisites and driver install/configuration MUST be performed on each
Docker host node!

### Golang
To install the latest Golang, just follow the easy steps on the Go install
websiste.  Again, I prefer to download the tarball and install myself rather
than use a package manager like apt or yum:
[Get Go](https://golang.org/doc/install)

NOTE:
It's very important that you follow the directions and setup your Go
environment.  After downloading the appropriate package be sure to scroll down
to the Linux, Mac OS X, and FreeBSD tarballs section and set up your Go
environment as per the instructions.

### Docker
As far as Docker install, again I prefer using wget and pulling the latest version
from get.docker.com.  You can find instructions and steps on the Docker website
here:
[Get Docker](https://docs.docker.com/linux/step_one/)

### Open iSCSI
This driver uses iSCSI SolidFire storage devices, and makes iSCSI connections
for your automatically.  In order to do that however you must have the Open
ISCSI packages installed on each Docker node.

- Open-iSCSI
  * Ubuntu<br>
  ```
  sudo apt-get install open-iscsi
  ```

  * Redhat variants<br>
  ```
  sud yum in stall iscsi-initiator-utils
  ```

## Driver Installation
### Download the linux binary from Github Release
```
wget https://github.com/solidfire/solidfire-docker-driver/releases/download/v1.1/solidfire-docker-driver

# move to a location in the bin path
sudo mv solidfire-docker-driver /usr/local/bin
sudo chown root:root /usr/local/bin/solidfire-docker-driver
```
### Build from source yourself
  ```
  go get -u github.com/solidfire/solidfire-docker-driver
  ```

** There are known issues with docker/go-plugins-helpers not building against the latest Docker version.  You can view the Issue on GitHub here:  https://github.com/docker/go-plugins-helpers/issues/46

SolidFire Github page [SolidFire Github](https://github.com/solidfire)

This will give you the source in your golang/src
The SolidFire plugin is made of multiple packages:
- The SolidFire Docker Service Daemon
- The SolidFire API (sfapi) modules providing our Go bindings
- The SolidFire admin CLI (sfcli) providing a CLI tool for administrative tasks
  including starting the daemon/service.

In addition to providing the source, this should also build and install the
solidfire-docker-driver binary in your Golang bin directory.

You will need to make sure you've added the $GOPATH/bin to your path,
AND on Ubuntu you will also need to enable the use of the GO Bin path by sudo;
either run visudo and edit, or provide an alias in your .bashrc file.

For example in your .bashrc set the following alias after setting up PATH:
  ```
  alias sudo='sudo env PATH=$PATH'
  ```

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
    "DefaultVolSz": 1,
    "InitiatorIFace": "default",
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

## Starting the daemon
After install and setting up a configuration, all you need to is start the
solidfire-docker-driver daemon so tha it can accept requests from Docker.

  ```
  sudo solidfire-docker-driver daemon start -v
  ```

## Usage Examples
Now that the daemon is running, you're ready to issue calls via the Docker
Volume API and have the requests serviced by the SolidFire Driver.

For a list of avaialable commands run:
  ```
  docker volume --help
  ```

Here's an example of how to create a SolidFire volume using the Docker Volume
API:
  ```
  docker volume create -d solidfire --name=testvolume
  ```

You can also specify options, like specifying the QoS via the Types you've set
in the config file:
  ```
  docker volume create -d solidfire --name=testvolume -o type=Gold
  -o size=10
  ```

Now in order to use that volume with a Container you simply specify
  ```
  docker run -v testvolume:/Data --volume-driver=solidfire -i -t ubuntu
  /bin/bash
  ```

Note that if you had NOT created the volume already, Docker will issue the
create call to the driver for you while launching the container.  The Driver
create method checks the SolidFire Cluster to see if the Volume already exists,
if it does it just passes back the info for the existing volume, otherwise it
runs through the create process and creates the Volume on the SolidFire
cluster.

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
