// Copyright (C) 2022-2023 Red Hat, Inc.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/sirupsen/logrus"
	"github.com/test-network-function/cnf-certification-test/pkg/configuration"
	corev1 "k8s.io/api/core/v1"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	plibRuntime "github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	plibContainer "github.com/redhat-openshift-ecosystem/openshift-preflight/container"
)

var (
	// Certain tests that have been known to fail because of injected containers (such as Istio) that fail certain tests.
	ignoredContainerNames = []string{"istio-proxy"}
)

type Container struct {
	*corev1.Container
	Status                   corev1.ContainerStatus
	Namespace                string
	Podname                  string
	NodeName                 string
	Runtime                  string
	UID                      string
	ContainerImageIdentifier configuration.ContainerImageIdentifier
	PreflightResults         plibRuntime.Results
}

func NewContainer() *Container {
	return &Container{
		Container: &corev1.Container{}, // initialize the corev1.Container object
	}
}

func (c *Container) GetUID() (string, error) {
	split := strings.Split(c.Status.ContainerID, "://")
	uid := ""
	if len(split) > 0 {
		uid = split[len(split)-1]
	}
	if uid == "" {
		logrus.Debugln(fmt.Sprintf("could not find uid of %s/%s/%s\n", c.Namespace, c.Podname, c.Name))
		return "", errors.New("cannot determine container UID")
	}
	logrus.Debugln(fmt.Sprintf("uid of %s/%s/%s=%s\n", c.Namespace, c.Podname, c.Name, uid))
	return uid, nil
}

func (c *Container) SetPreflightResults(preflightImageCache map[string]plibRuntime.Results, env *TestEnvironment) error {
	logrus.Infof("Running preflight container test against image: %s with name: %s", c.Image, c.Name)

	// Short circuit if the image already exists in the cache
	if _, exists := preflightImageCache[c.Image]; exists {
		logrus.Infof("Container image: %s exists in the cache. Skipping this run.", c.Image)
		c.PreflightResults = preflightImageCache[c.Image]
		return nil
	}

	opts := []plibContainer.Option{}
	opts = append(opts, plibContainer.WithDockerConfigJSONFromFile(env.GetDockerConfigFile()))
	if env.IsPreflightInsecureAllowed() {
		logrus.Info("Insecure connections are being allowed to preflight")
		opts = append(opts, plibContainer.WithInsecureConnection())
	}

	// Create artifacts handler
	artifactsWriter, err := artifacts.NewMapWriter()
	if err != nil {
		return err
	}
	ctx := artifacts.ContextWithWriter(context.TODO(), artifactsWriter)

	// Add logger output to the context
	logbytes := bytes.NewBuffer([]byte{})
	checklogger := log.Default()
	checklogger.SetOutput(logbytes)
	logger := stdr.New(checklogger)
	ctx = logr.NewContext(ctx, logger)

	check := plibContainer.NewCheck(c.Image, opts...)
	results, runtimeErr := check.Run(ctx)
	logrus.StandardLogger().Out = os.Stderr
	if runtimeErr != nil {
		logrus.Error(runtimeErr)
		return runtimeErr
	}

	// Take all of the preflight logs and stick them into logrus.
	logrus.Info(logbytes.String())

	// Store the result into the cache and store the Results into the container's PreflightResults var.
	preflightImageCache[c.Image] = results
	c.PreflightResults = preflightImageCache[c.Image]
	return nil
}

func (c *Container) StringLong() string {
	return fmt.Sprintf("node: %s ns: %s podName: %s containerName: %s containerUID: %s containerRuntime: %s",
		c.NodeName,
		c.Namespace,
		c.Podname,
		c.Name,
		c.Status.ContainerID,
		c.Runtime,
	)
}
func (c *Container) String() string {
	return fmt.Sprintf("container: %s pod: %s ns: %s",
		c.Name,
		c.Podname,
		c.Namespace,
	)
}

func (c *Container) HasIgnoredContainerName() bool {
	for _, ign := range ignoredContainerNames {
		if c.IsIstioProxy() || strings.Contains(c.Name, ign) {
			return true
		}
	}
	return false
}

func (c *Container) IsIstioProxy() bool {
	return c.Name == "istio-proxy" //nolint:goconst
}

func (c *Container) HasExecProbes() bool {
	return c.LivenessProbe != nil && c.LivenessProbe.Exec != nil ||
		c.ReadinessProbe != nil && c.ReadinessProbe.Exec != nil ||
		c.StartupProbe != nil && c.StartupProbe.Exec != nil
}

func (c *Container) IsTagEmpty() bool {
	return c.ContainerImageIdentifier.Tag == ""
}
