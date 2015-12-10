// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client_test

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/names"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"

	basetesting "github.com/juju/juju/api/base/testing"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/resource"
	"github.com/juju/juju/resource/api"
	"github.com/juju/juju/resource/api/client"
)

var _ = gc.Suite(&SpecSuite{})

type SpecSuite struct {
	testing.IsolationSuite

	stub    *testing.Stub
	facade  *stubFacade
	apiSpec api.ResourceSpec
}

func (s *SpecSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.stub = &testing.Stub{}
	s.facade = newStubFacade(c, s.stub)
	s.apiSpec = api.ResourceSpec{
		Name:     "spam",
		Type:     "file",
		Path:     "spam.tgz",
		Comment:  "you need it",
		Origin:   "upload",
		Revision: "",
	}
}

func (s *SpecSuite) TestListSpecsOkay(c *gc.C) {
	expected, apiResult := newSpecResult(c, "a-service", "spam")
	s.facade.apiResults["a-service"] = apiResult

	cl := client.NewClient(s.facade)

	services := []string{"a-service"}
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, jc.DeepEquals, []resource.SpecsResult{
		expected,
	})
	c.Check(s.stub.Calls(), gc.HasLen, 1)
	s.stub.CheckCall(c, 0, "FacadeCall",
		"ListSpecs",
		&api.ListSpecsArgs{
			Entities: []params.Entity{{
				Tag: "service-a-service",
			}},
		},
		&api.ResourceSpecsResults{
			Results: []api.ResourceSpecsResult{
				apiResult,
			},
		},
	)
}

func (s *SpecSuite) TestListSpecsBulk(c *gc.C) {
	expected1, apiResult1 := newSpecResult(c, "a-service", "spam")
	s.facade.apiResults["a-service"] = apiResult1
	expected2, apiResult2 := newSpecResult(c, "other-service", "eggs", "ham")
	s.facade.apiResults["other-service"] = apiResult2

	cl := client.NewClient(s.facade)

	services := []string{"a-service", "other-service"}
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, jc.DeepEquals, []resource.SpecsResult{
		expected1,
		expected2,
	})
	c.Check(s.stub.Calls(), gc.HasLen, 1)
	s.stub.CheckCall(c, 0, "FacadeCall",
		"ListSpecs",
		&api.ListSpecsArgs{
			Entities: []params.Entity{{
				Tag: "service-a-service",
			}, {
				Tag: "service-other-service",
			}},
		},
		&api.ResourceSpecsResults{
			Results: []api.ResourceSpecsResult{
				apiResult1,
				apiResult2,
			},
		},
	)
}

func (s *SpecSuite) TestListSpecsNoServices(c *gc.C) {
	cl := client.NewClient(s.facade)

	var services []string
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, gc.HasLen, 0)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsBadServices(c *gc.C) {
	cl := client.NewClient(s.facade)

	services := []string{"???"}
	_, err := cl.ListSpecs(services)

	c.Check(err, gc.ErrorMatches, `.*invalid service.*`)
	s.stub.CheckNoCalls(c)
}

func (s *SpecSuite) TestListSpecsServiceNotFound(c *gc.C) {
	cl := client.NewClient(s.facade)

	services := []string{"a-service"}
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, jc.DeepEquals, []resource.SpecsResult{{
		Service: "a-service",
		Error:   results[0].Error,
	}})
	c.Check(results[0].Error, jc.Satisfies, errors.IsNotFound)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsServiceEmpty(c *gc.C) {
	s.facade.apiResults["a-service"] = api.ResourceSpecsResult{}

	cl := client.NewClient(s.facade)

	services := []string{"a-service"}
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, jc.DeepEquals, []resource.SpecsResult{{
		Service: "a-service",
	}})
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsServerError(c *gc.C) {
	failure := errors.New("<failure>")
	s.facade.FacadeCallFn = func(_ string, _, _ interface{}) error {
		return failure
	}

	cl := client.NewClient(s.facade)

	services := []string{"a-service"}
	_, err := cl.ListSpecs(services)

	c.Check(err, gc.ErrorMatches, `<failure>`)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsTooFew(c *gc.C) {
	s.facade.FacadeCallFn = func(_ string, _, response interface{}) error {
		typedResponse, ok := response.(*api.ResourceSpecsResults)
		c.Assert(ok, jc.IsTrue)

		typedResponse.Results = []api.ResourceSpecsResult{{
			Specs: nil,
		}}

		return nil
	}

	cl := client.NewClient(s.facade)

	services := []string{"a-service", "other-service"}
	results, err := cl.ListSpecs(services)

	c.Check(results, gc.HasLen, 0)
	c.Check(err, gc.ErrorMatches, `.*got invalid data from server \(expected 2 results, got 1\).*`)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsTooMany(c *gc.C) {
	s.facade.FacadeCallFn = func(_ string, _, response interface{}) error {
		typedResponse, ok := response.(*api.ResourceSpecsResults)
		c.Assert(ok, jc.IsTrue)

		typedResponse.Results = []api.ResourceSpecsResult{{
			Specs: nil,
		}, {
			Specs: nil,
		}, {
			Specs: nil,
		}}

		return nil
	}

	cl := client.NewClient(s.facade)

	services := []string{"a-service", "other-service"}
	results, err := cl.ListSpecs(services)

	c.Check(results, gc.HasLen, 0)
	c.Check(err, gc.ErrorMatches, `.*got invalid data from server \(expected 2 results, got 3\).*`)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func (s *SpecSuite) TestListSpecsConversionFailed(c *gc.C) {
	s.facade.FacadeCallFn = func(_ string, _, response interface{}) error {
		typedResponse, ok := response.(*api.ResourceSpecsResults)
		c.Assert(ok, jc.IsTrue)

		typedResponse.Results = []api.ResourceSpecsResult{{
			Specs: []api.ResourceSpec{{
				Name: "spam",
			}},
		}}

		return nil
	}

	cl := client.NewClient(s.facade)

	services := []string{"a-service"}
	results, err := cl.ListSpecs(services)
	c.Assert(err, jc.ErrorIsNil)

	c.Check(results, jc.DeepEquals, []resource.SpecsResult{{
		Service: "a-service",
		Specs: []resource.Spec{{
			Definition: charmresource.Info{
				Name: "spam",
			},
		}},
		Error: results[0].Error,
	}})
	c.Check(results[0].Error, gc.ErrorMatches, `.*got bad data.*`)
	s.stub.CheckCallNames(c, "FacadeCall")
}

func newSpecResult(c *gc.C, serviceID string, names ...string) (resource.SpecsResult, api.ResourceSpecsResult) {
	result := resource.SpecsResult{
		Service: serviceID,
	}
	var apiResult api.ResourceSpecsResult
	for _, name := range names {
		spec, apiSpec := newSpec(c, name)
		result.Specs = append(result.Specs, spec)
		apiResult.Specs = append(apiResult.Specs, apiSpec)
	}
	return result, apiResult
}

func newSpec(c *gc.C, name string) (resource.Spec, api.ResourceSpec) {
	spec := resource.Spec{
		Definition: charmresource.Info{
			Name: name,
			Type: charmresource.TypeFile,
			Path: name + ".tgz",
		},
		Origin:   resource.OriginKindUpload,
		Revision: resource.NoRevision,
	}
	err := spec.Validate()
	c.Assert(err, jc.ErrorIsNil)

	apiSpec := api.ResourceSpec{
		Name:     name,
		Type:     "file",
		Path:     name + ".tgz",
		Comment:  "",
		Origin:   "upload",
		Revision: "",
	}

	return spec, apiSpec
}

type stubFacade struct {
	basetesting.StubFacadeCaller

	apiResults map[string]api.ResourceSpecsResult
}

func newStubFacade(c *gc.C, stub *testing.Stub) *stubFacade {
	s := &stubFacade{
		StubFacadeCaller: basetesting.StubFacadeCaller{
			Stub: stub,
		},
		apiResults: make(map[string]api.ResourceSpecsResult),
	}

	s.FacadeCallFn = func(_ string, args, response interface{}) error {
		typedResponse, ok := response.(*api.ResourceSpecsResults)
		c.Assert(ok, jc.IsTrue)

		typedArgs, ok := args.(*api.ListSpecsArgs)
		c.Assert(ok, jc.IsTrue)

		for _, e := range typedArgs.Entities {
			tag, err := names.ParseTag(e.Tag)
			c.Assert(err, jc.ErrorIsNil)
			service := tag.Id()

			apiResult, ok := s.apiResults[service]
			if !ok {
				apiResult.Error = &params.Error{
					Message: fmt.Sprintf("service %q not found", service),
					Code:    params.CodeNotFound,
				}
			}
			typedResponse.Results = append(typedResponse.Results, apiResult)
		}
		return nil
	}

	return s
}

func (s *stubFacade) Close() error {
	s.Stub.AddCall("Close")
	if err := s.Stub.NextErr(); err != nil {
		return errors.Trace(err)
	}

	return nil
}