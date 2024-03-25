package cmd

import (
	"strings"

	"github.com/jumppad-labs/jumppad/cmd/changelog"
	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show the changelog",
	Long:  `Show the changelog`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cl := &changelog.Changelog{}

		// replace """ with ``` in changelog
		changes = strings.ReplaceAll(changes, `"""`, "```")

		err := cl.Show(changes, changesVersion, true)
		if err != nil {
			showErr(err)
		}
	},
}

var changesVersion = "v0.10.0"

var changes = `
## version v0.10.1
* Ensure that the total compute is set correctly	for Nomad clusters when 
	running on Docker in Apple Silicon.

## version v0.10.0

## New Features:
* Add experimental cancellation for long running commands, you can
  now press 'ctrl-c' to interupt 'up' and 'down' commands
* Add --force flag to ignore graceful exit for the down command

### Breaking Changes: 

#### Exec Local and Exec Remote Resources
The 'exec_local' and 'exec_remote' resources have been removed in favor
of the new 'exec' resource. The 'exec' resource supports all the functionality
of the old resources and more.

"""hcl
resource "container" "alpine" {
  image {
    name = "alpine"
  }

  command = ["tail", "-f", "/dev/null"]

  volume {
    source      = data("test")
    destination = "/data"
  }
}

resource "exec" "run" {
  script = <<-EOF
  #!/bin/sh
  ${data("test")}/consul agent -dev
  EOF

  daemon = true
}
"""

#### Kubernetes Cluster Configuration

Prior to this version Kubernetes clusters could access the config path like
the following example:

"""
resource "k8s_cluster" "k3s" {
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}
"""

In the latest version this has changed to expand the details of the kubeconfig
providing access to the cluster ca certificate, client certificate and client key.

An updated example can be seen below:

"""
resource "k8s_cluster" "k3s" {
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kube_config.path
}

output "KUBE_CA" {
  value = resource.k8s_cluster.k3s.kube_config.ca
}

output "KUBE_CLIENT_CERT" {
  value = resource.k8s_cluster.k3s.kube_config.client_certificate
}

output "KUBE_CLIENT_KEY" {
  value = resource.k8s_cluster.k3s.kube_config.client_key
}
"""

## version v0.9.1
* Update internal references to use the new 'local.jmpd.in' domain bypassing
  problems where chrome auto redirects .dev to https://.
* Update Nomad to 1.7.5

## version v0.7.0

### Breaking Changes:
This version of Jumppad introduces experimental plugin support for custom resources. 
To avoid conflicts between the default properties and the custom properties 
the default properties for a resource have been renamed to prefix "resource_" 
to their name. For example previously to reference the "id" of a resource you could write:

"""
resource.container.mine.id
"""

This has now changed to:

"""
resource.container.mine.meta.id
"""

From this version onwards the old property names are no longer be supported 
and you may need to update your configuration.

The full list of properties tha have been changed are:

| Old Property Name | New Property Name   |
|-------------------|---------------------|
| id                | meta.id         |
| name              | meta.name       |
| type              | meta.type       |
| module            | resource_module     |
| file              | resource_file       |
| line              | resource_line       |
| column            | resource_column     |
| checksum          | resource_checksum   |
| checksum          | resource_checksum   |
| properties        | resource_properties |

### Features:
* Add capability to add custom container registries to the image cache  

Nomad and Kuberentes clusters are started in a Docker container that does not save any state to the local disk.
This state includes and Docker Image cache, thefore every time an image is pulled to a new cluster it is downloaded
from the internet. This can be slow and bandwidth intensive. To solve this problem Jumppad implemented a pull through
cache that is used by all clusters. By default this cache supported the following registires:  

  - k8s.gcr.io 
  - gcr.io 
  - asia.gcr.io
  - eu.gcr.io
  - us.gcr.io 
  - quay.io
  - ghcr.io
  - docker.pkg.github.com
  
To support custom registries Jumppad has added a new resource type "container_registry". This resource type can be used
to define either a local or remote registry. When a registry is defined it is added to the pull through cache and
any authnetication details are added to the cache meaning you do not need to authenticate each pull on the Nomad or 
Kubernetes cluster. Any defined registry must be configured to use HTTPS, the image cache can not be used to pull
from insecure registries.

"""hcl
# Define a custom registry that does not use authentication
resource "container_registry" "noauth" {
  hostname = "noauth-registry.demo.gs" // cache can not resolve local.jmpd.in dns for some reason, 
                                       // using external dns mapped to the local ip address
}

# Define a custom registry that uses authentication
resource "container_registry" "auth" {
  hostname = "auth-registry.demo.gs"
  auth {
    username = "admin"
    password = "password"
  }
}
"""

* Add capability to add insecure registries and image cache bypass to Kubernetes and Nomad clusters.
  
All images pulled to Nomad and Kubernetes clusters are pulled through the image cache. This cache is a Docker
container that is automatically started by Jumppad. To disable the cache and pull images directly from the internet
you can add the "no_proxy" parameter to the new docker config stanza. This will cause the cache to be bypassed and
the image to be pulled direct from the internet.  

To support insecure registries you can add the "insecure_registries" parameter to the docker config stanza. This
must be used in conjunction with the "no_proxy" parameter as the image cache does not support insecure registries. 

"""hcl
resource "nomad_cluster" "dev" {
  client_nodes = 1

  datacenter = "dc1"

  network {
    id = variable.network_id
  }

  // add configuration to allow cache bypass and insecure registry
  config {
    docker {
      no_proxy            = ["insecure.container.jmpd.in"]
      insecure_registries = ["insecure.container.jmpd.in:5003"]
    }
  }
}
"""

## version v0.5.47
* Fix isuse where filepath.Walk does not respect symlinks
* Add "ignore" parameter to "build" resource to allow ignoring of files and folders
  for Docker builds.
	`
