package providers

import (
	"fmt"
	"net"

	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"golang.org/x/xerrors"
)

// Ingress defines a provider for handling connection ingress for a cluster
type Ingress struct {
	config    *resources.Ingress
	client    clients.ContainerTasks
	connector clients.Connector
	log       clients.Logger
}

// NewIngress creates a new ingress provider
func NewIngress(
	c *resources.Ingress,
	cc clients.ContainerTasks,
	co clients.Connector,
	l clients.Logger) *Ingress {

	return &Ingress{c, cc, co, l}
}

func (c *Ingress) Create() error {
	c.log.Info("Create Ingress", "ref", c.config.ID)

	return c.exposeRemote()
	//if c.config.Destination.Driver == "local" {
	//}

	//if c.config.Destination.Driver == "k8s" {
	//	return c.exposeK8sRemote()
	//}
}

// Destroy satisfies the interface method but is not implemented by LocalExec
func (c *Ingress) Destroy() error {
	c.log.Info("Destroy Ingress", "ref", c.config.ID, "id", c.config.IngressID)

	err := c.connector.RemoveService(c.config.IngressID)
	if err != nil {
		// fail silently as this should not stop us from destroying the
		// other resources
		c.log.Warn("Unable to remove local ingress", "ref", c.config.Name, "id", c.config.IngressID, "error", err)
	}

	return nil
}

// Lookup satisfies the interface method but is not implemented by LocalExec
func (c *Ingress) Lookup() ([]string, error) {
	c.log.Debug("Lookup Ingress", "ref", c.config.ID, "id", c.config.IngressID)

	return []string{}, nil
}

func (c *Ingress) Refresh() error {
	c.log.Debug("Refresh Ingress", "ref", c.config.Name)

	return nil
}

func (c *Ingress) Changed() (bool, error) {
	c.log.Debug("Checking changes", "ref", c.config.Name)

	return false, nil
}

func (c *Ingress) exposeRemote() error {
	// get the target
	r, err := c.config.ParentConfig.FindResource(c.config.Target.ID)
	if err != nil {
		return err
	}

	// check if the port is in use, if so, return an immediate error
	c.log.Debug("Checking if port is available", "port", c.config.Port)
	tc, err := net.Dial("tcp", fmt.Sprintf("0.0.0.0:%d", c.config.Port))
	if err == nil {
		c.log.Debug("Port in use", "port", c.config.Port)
		return fmt.Errorf("unable to create ingress port %d in use", c.config.Port)
	}

	if tc != nil {
		tc.Close()
	}

	// address of the remote connector
	connectorAddress := ""

	// destination address depends on the type of the cluster
	destAddr := ""
	port := fmt.Sprintf("%d", c.config.Target.Port)

	if c.config.Target.NamedPort != "" {
		port = c.config.Target.NamedPort
	}

	switch r.Metadata().Type {
	case resources.TypeK8sCluster:
		destAddr = fmt.Sprintf(
			"%s.%s.svc:%s",
			c.config.Target.Config["service"],
			c.config.Target.Config["namespace"],
			port,
		)

		k8s := r.(*resources.K8sCluster)
		connectorAddress = fmt.Sprintf("%s:%d", k8s.ExternalIP, k8s.ConnectorPort)

	case resources.TypeNomadCluster:
		destAddr = fmt.Sprintf(
			"%s.%s.%s:%s",
			c.config.Target.Config["job"],
			c.config.Target.Config["group"],
			c.config.Target.Config["task"],
			port,
		)

		n3d := r.(*resources.NomadCluster)
		connectorAddress = fmt.Sprintf("%s:%d", n3d.ExternalIP, n3d.ConnectorPort)
	}

	// sanitize the name to make it uri format
	serviceName, err := utils.ReplaceNonURIChars(c.config.Name)
	if err != nil {
		return xerrors.Errorf("unable to replace non URI characters in service name %s :%w", c.config.Name, err)
	}

	// send the request
	c.log.Debug(
		"Calling connector to expose local service",
		"name", serviceName,
		"local_port", c.config.Port,
		"connector_addr", connectorAddress,
		"remote_addr", destAddr,
	)

	id, err := c.connector.ExposeService(
		serviceName,
		c.config.Port,
		connectorAddress,
		destAddr,
		"remote",
	)

	if err != nil {
		return xerrors.Errorf("unable to expose remote service on cluster :%w", err)
	}

	addr := fmt.Sprintf("%s:%d", utils.GetDockerIP(), c.config.Port)
	c.log.Debug("Successfully exposed service", "id", id, "dest", destAddr, "addr", addr)

	c.config.IngressID = id
	c.config.Address = addr

	return nil
}

// exposeK8sRemote exposes a remote kubernetes service to the local machine
//func (c *Ingress) exposeK8sRemote() error {
//	// get the target
//	res, err := c.config.ParentConfig.FindResource(c.config.Destination.Config.Cluster)
//	if err != nil {
//		return err
//	}
//
//	if c.config.Destination.Config.Address == "" {
//		return xerrors.Errorf("config parameter 'address' is required for destinations of type 'k8s'")
//	}
//
//	destAddr := fmt.Sprintf("%s:%s", c.config.Destination.Config.Address, c.config.Destination.Config.Port)
//
//	localPort, err := strconv.Atoi(c.config.Source.Config.Port)
//	if err != nil {
//		return xerrors.Errorf("Unable to parse remote port :%w", err)
//	}
//
//	if localPort == 30001 || localPort == 30002 {
//		return fmt.Errorf("unable to expose local service using remote port %d,"+
//			"ports 30001 and 30002 are reserved for internal use", localPort)
//	}
//
//	// sanitize the name to make it uri format
//	serviceName, err := utils.ReplaceNonURIChars(c.config.Name)
//	if err != nil {
//		return xerrors.Errorf("unable to replace non URI characters in service name %s :%w", c.config.Name, err)
//	}
//
//	connectorAddress := fmt.Sprintf("%s:%d", res.(*resources.K8sCluster).ExternalIP, res.(*resources.K8sCluster).ConnectorPort)
//
//	// send the request
//	c.log.Debug(
//		"Calling connector to expose remote service",
//		"name", serviceName,
//		"local_port", localPort,
//		"connector_addr", connectorAddress,
//		"local_addr", destAddr,
//	)
//
//	id, err := c.connector.ExposeService(
//		serviceName,
//		localPort,
//		connectorAddress,
//		destAddr,
//		"remote")
//
//	if err != nil {
//		return xerrors.Errorf("unable to expose remote cluster service to local machine :%w", err)
//	}
//
//	local, _ := utils.GetLocalIPAndHostname()
//	addr := fmt.Sprintf("%s:%d", local, localPort)
//
//	c.log.Debug("Successfully exposed service", "id", id, "addr", addr)
//
//	c.config.IngressID = id
//	c.config.Address = addr
//
//	return nil
//}
