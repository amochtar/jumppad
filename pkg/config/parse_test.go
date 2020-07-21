package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func setup() func() {
	os.Setenv("SHIPYARD_CONFIG", "/User/yamcha/.shipyard")

	return func() {
		os.Unsetenv("SHIPYARD_CONFIG")
	}
}

func TestRunParsesBlueprintInMarkdownFormat(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/container")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	assert.NotNil(t, c.Blueprint)

	assert.Equal(t, "Nic Jackson", c.Blueprint.Author)
	assert.Equal(t, "Single Container Example", c.Blueprint.Title)
	assert.Equal(t, "container", c.Blueprint.Slug)
	assert.Equal(t, []string{"http://consul-http.ingress.shipyard.run:8500"}, c.Blueprint.BrowserWindows)
	assert.Equal(t, "SOMETHING", c.Blueprint.Environment[0].Key)
	assert.Equal(t, "else", c.Blueprint.Environment[0].Value)
	assert.Contains(t, c.Blueprint.Intro, "# Single Container")
	assert.Contains(t, c.Blueprint.HealthCheckTimeout, "30s")
}

func TestRunParsesBlueprintInHCLFormat(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/single_k3s_cluster")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	assert.NotNil(t, c.Blueprint)
}

func TestLoadsVariablesFiles(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/container")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	// check variable has been interpolated
	r, err := c.FindResource("container.consul")
	assert.NoError(t, err)

	validEnv := false
	con := r.(*Container)
	for _, e := range con.Environment {
		// should contain a key called "something" with a value "else"
		if e.Key == "something" && e.Value == "blah blah" {
			validEnv = true
		}
	}

	assert.True(t, validEnv)
}

func TestOverridesVariablesFilesWithFlag(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/container")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, map[string]string{"something": "else"})
	assert.NoError(t, err)

	// check variable has been interpolated
	r, err := c.FindResource("container.consul")
	assert.NoError(t, err)

	validEnv := false
	con := r.(*Container)
	for _, e := range con.Environment {
		// should contain a key called "something" with a value "else"
		if e.Key == "something" && e.Value == "else" {
			validEnv = true
		}
	}

	assert.True(t, validEnv)
}

func TestOverridesVariablesFilesWithEnv(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/container")
	if err != nil {
		t.Fatal(err)
	}

	os.Setenv("SY_VAR_something", "env")
	t.Cleanup(func() {
		os.Unsetenv("SY_VAR_something")
	})

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	// check variable has been interpolated
	r, err := c.FindResource("container.consul")
	assert.NoError(t, err)

	validEnv := false
	con := r.(*Container)
	for _, e := range con.Environment {
		// should contain a key called "something" with a value "else"
		if e.Key == "something" && e.Value == "env" {
			validEnv = true
		}
	}

	assert.True(t, validEnv)
}

func TestDoesNotLoadsVariablesFilesFromInsideModules(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/modules")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	// check variable has been interpolated
	r, err := c.FindResource("container.consul")
	assert.NoError(t, err)

	validEnv := false
	con := r.(*Container)
	for _, e := range con.Environment {
		fmt.Println(e.Value)
		// should contain a key called "something" with a value "else"
		if e.Key == "something" && e.Value == "this is a module" {
			validEnv = true
		}
	}

	assert.True(t, validEnv)
}

func TestParseModuleCreatesResources(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/modules")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	// count the resources, should create 10
	assert.Len(t, c.Resources, 13)

	// check depends on is set
	r, err := c.FindResource("k8s_cluster.k3s")
	assert.NoError(t, err)
	assert.Contains(t, r.Info().DependsOn, "module.consul")

	// check the module is set on resources loaded as a module
	r, err = c.FindResource("container.consul")
	assert.NoError(t, err)
	assert.Equal(t, "consul", r.Info().Module)
}

func TestParseFileFunctionReadCorrectly(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("../../examples/container")
	if err != nil {
		t.Fatal(err)
	}

	c := New()
	err = ParseFolder(absoluteFolderPath, c, false, nil)
	assert.NoError(t, err)

	// check variable has been interpolated
	r, err := c.FindResource("container.consul")
	assert.NoError(t, err)

	validEnv := false
	con := r.(*Container)
	for _, e := range con.Environment {
		// should contain a key called "something" with a value "else"
		if e.Key == "file" && e.Value == "this is the contents of a file" {
			validEnv = true
		}
	}

	assert.True(t, validEnv)
}

/*
func TestSingleKubernetesCluster(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("./examples/single-cluster-k8s")
	if err != nil {
		t.Fatal(err)
	}

	tearDown := setup()
	defer tearDown()

	c := New()
	err = ParseFolder("./examples/single-cluster-k8s", c)

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// validate clusters
	assert.Len(t, c.Clusters, 1)

	c1 := c.Clusters[0]
	assert.Equal(t, "default", c1.Name)
	assert.Equal(t, "1.16.0", c1.Version)
	assert.Equal(t, 3, c1.Nodes)
	assert.Equal(t, "network.k8s", c1.Network)

	// validate networks
	assert.Len(t, c.Networks, 1)

	n1 := c.Networks[0]
	assert.Equal(t, "k8s", n1.Name)
	assert.Equal(t, "10.4.0.0/16", n1.Subnet)

	// validate helm charts
	assert.Len(t, c.HelmCharts, 1)

	h1 := c.HelmCharts[0]
	assert.Equal(t, "cluster.default", h1.Cluster)
	assert.Equal(t, "/User/yamcha/.shipyard/charts/consul", h1.Chart)
	assert.Equal(t, fmt.Sprintf("%s/consul-values", absoluteFolderPath), h1.Values)
	assert.Equal(t, "component=server,app=consul", h1.HealthCheck.Pods[0])
	assert.Equal(t, "component=client,app=consul", h1.HealthCheck.Pods[1])

	// validate ingress
	assert.Len(t, c.Ingresses, 2)

	i1 := c.Ingresses[0]
	assert.Equal(t, "consul", i1.Name)
	assert.Equal(t, 8500, i1.Ports[0].Local)
	assert.Equal(t, 8500, i1.Ports[0].Remote)
	assert.Equal(t, 8500, i1.Ports[0].Host)

	i2 := c.Ingresses[1]
	assert.Equal(t, "web", i2.Name)

	// validate references
	err = ParseReferences(c)
	assert.NoError(t, err)

	assert.Equal(t, n1, c1.NetworkRef)
	assert.Equal(t, c.WAN, c1.WANRef)
	assert.Equal(t, c1, h1.ClusterRef)
	assert.Equal(t, i1.TargetRef, c1)
	assert.Equal(t, i1.NetworkRef, n1)
	assert.Equal(t, c.WAN, i1.WANRef)
	assert.Equal(t, i2.TargetRef, c1)
}

func TestMultiCluster(t *testing.T) {
	absoluteFolderPath, err := filepath.Abs("./examples/multi-cluster")
	if err != nil {
		t.Fatal(err)
	}

	tearDown := setup()
	defer tearDown()

	c := New()
	err = ParseFolder(absoluteFolderPath, c)

	assert.NoError(t, err)
	assert.NotNil(t, c)

	// validate clusters
	assert.Len(t, c.Clusters, 2)

	c1 := c.Clusters[0]
	assert.Equal(t, "cloud", c1.Name)
	assert.Equal(t, "1.16.0", c1.Version)
	assert.Equal(t, 1, c1.Nodes)
	assert.Equal(t, "network.k8s", c1.Network)

	// validate containers
	assert.Len(t, c.Containers, 2)

	co1 := c.Containers[0]
	assert.Equal(t, "consul_nomad", co1.Name)
	assert.Equal(t, []string{"consul", "agent", "-config-file=/config/consul.hcl"}, co1.Command)
	assert.Equal(t, fmt.Sprintf("%s/consul_config", absoluteFolderPath), co1.Volumes[0].Source, "Volume should have been converted to be absolute")
	assert.Equal(t, "/config", co1.Volumes[0].Destination)
	assert.Equal(t, "network.nomad", co1.Network)
	assert.Equal(t, "10.6.0.2", co1.IPAddress)

	// validate ingress
	assert.Len(t, c.Ingresses, 6)

	i1 := testFindIngress("consul_nomad", c.Ingresses)
	assert.Equal(t, "consul_nomad", i1.Name)

	// validate references
	err = ParseReferences(c)
	assert.NoError(t, err)

	assert.Equal(t, co1, i1.TargetRef)
	assert.Equal(t, c.WAN, i1.WANRef)

	// validate documentation
	d1 := c.Docs
	assert.Equal(t, "multi-cluster", d1.Name)
	assert.Equal(t, fmt.Sprintf("%s/docs", absoluteFolderPath), d1.Path)
	assert.Equal(t, 8080, d1.Port)
	assert.Equal(t, "index.html", d1.Index)
}

func testFindIngress(name string, ingress []*Ingress) *Ingress {
	for _, i := range ingress {
		if i.Name == name {
			return i
		}
	}

	return nil
}
*/
