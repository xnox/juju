// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuclienttesting

import (
	"github.com/juju/errors"

	"github.com/juju/juju/jujuclient"
)

type inMemory struct {
	all map[string]jujuclient.Controller
}

func NewMem() jujuclient.Cache {
	return &inMemory{make(map[string]jujuclient.Controller)}
}

// AllControllers implements ControllersGetter.AllControllers
func (c *inMemory) AllControllers() (map[string]jujuclient.Controller, error) {
	return c.all, nil
}

// ControllerByName implements ControllersGetter.ControllerByName
func (c *inMemory) ControllerByName(name string) (*jujuclient.Controller, error) {
	if result, ok := c.all[name]; ok {
		return &result, nil
	}
	return nil, errors.NotFoundf("controller %s", name)
}

// UpdateController implements ControllersUpdater.UpdateController
func (c *inMemory) UpdateController(name string, one jujuclient.Controller) error {
	if err := jujuclient.ValidateControllerDetails(name, one); err != nil {
		return err
	}
	c.all[name] = one
	return nil
}

// RemoveController implements ControllersRemover.RemoveController
func (c *inMemory) RemoveController(name string) error {
	delete(c.all, name)
	return nil
}
