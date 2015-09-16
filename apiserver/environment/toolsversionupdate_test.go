// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package environment

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state"
	coretesting "github.com/juju/juju/testing"
	coretools "github.com/juju/juju/tools"
	"github.com/juju/juju/version"
)

var _ = gc.Suite(&updaterSuite{})

type updaterSuite struct {
	coretesting.BaseSuite
}

type dummyEnviron struct {
	environs.Environ
}

// SampleConfig() returns an environment configuration with all required
// attributes set.
func sampleConfig() coretesting.Attrs {
	return coretesting.Attrs{
		"type":                      "dummy",
		"name":                      "only",
		"uuid":                      coretesting.EnvironmentTag.Id(),
		"authorized-keys":           coretesting.FakeAuthKeys,
		"firewall-mode":             config.FwInstance,
		"admin-secret":              coretesting.DefaultMongoPassword,
		"ca-cert":                   coretesting.CACert,
		"ca-private-key":            coretesting.CAKey,
		"ssl-hostname-verification": true,
		"development":               false,
		"state-port":                1234,
		"api-port":                  4321,
		"syslog-port":               2345,
		"default-series":            config.LatestLtsSeries(),

		"secret":       "pork",
		"state-server": true,
		"prefer-ipv6":  true,
	}
}

func (s *updaterSuite) TestCheckTools(c *gc.C) {
	sConfig := sampleConfig()
	sConfig["agent-version"] = "2.5.0"
	cfg, err := config.New(config.NoDefaults, sConfig)
	c.Assert(err, jc.ErrorIsNil)
	fakeNewEnvirons := func(*config.Config) (environs.Environ, error) {
		return dummyEnviron{}, nil
	}
	s.PatchValue(&newEnvirons, fakeNewEnvirons)
	var (
		calledWithEnviron                environs.Environ
		calledWithMajor, calledWithMinor int
		calledWithFilter                 coretools.Filter
	)
	fakeToolFinder := func(e environs.Environ, maj int, min int, filter coretools.Filter) (coretools.List, error) {
		calledWithEnviron = e
		calledWithMajor = maj
		calledWithMinor = min
		calledWithFilter = filter
		ver := version.Binary{Number: version.Number{Major: maj, Minor: min}}
		t := coretools.Tools{Version: ver, URL: "http://example.com", Size: 1}
		c.Assert(calledWithMajor, gc.Equals, 2)
		c.Assert(calledWithMinor, gc.Equals, 5)
		return coretools.List{&t}, nil
	}

	ver, err := checkToolsAvailability(cfg, fakeToolFinder)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(ver, gc.Not(gc.Equals), version.Zero)
	c.Assert(ver, gc.Equals, version.Number{Major: 2, Minor: 5, Patch: 0})
}

type envGetter struct {
}

func (e *envGetter) Environment() (*state.Environment, error) {
	return &state.Environment{}, nil
}

func (s *updaterSuite) TestUpdateToolsAvailability(c *gc.C) {
	fakeNewEnvirons := func(*config.Config) (environs.Environ, error) {
		return dummyEnviron{}, nil
	}
	s.PatchValue(&newEnvirons, fakeNewEnvirons)

	fakeEnvConfig := func(_ *state.Environment) (*config.Config, error) {
		sConfig := sampleConfig()
		sConfig["agent-version"] = "2.5.0"
		return config.New(config.NoDefaults, sConfig)
	}
	s.PatchValue(&envConfig, fakeEnvConfig)

	fakeToolFinder := func(_ environs.Environ, _ int, _ int, _ coretools.Filter) (coretools.List, error) {
		ver := version.Binary{Number: version.Number{Major: 2, Minor: 5, Patch: 2}}
		olderVer := version.Binary{Number: version.Number{Major: 2, Minor: 5, Patch: 1}}
		t := coretools.Tools{Version: ver, URL: "http://example.com", Size: 1}
		tOld := coretools.Tools{Version: olderVer, URL: "http://example.com", Size: 1}
		return coretools.List{&t, &tOld}, nil
	}

	var ver version.Number
	fakeUpdate := func(_ *state.Environment, v version.Number) error {
		ver = v
		return nil
	}

	err := updateToolsAvailability(&envGetter{}, fakeToolFinder, fakeUpdate)
	c.Assert(err, jc.ErrorIsNil)

	c.Assert(ver, gc.Not(gc.Equals), version.Zero)
	c.Assert(ver, gc.Equals, version.Number{Major: 2, Minor: 5, Patch: 2})
}