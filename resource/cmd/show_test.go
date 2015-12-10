// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package cmd_test

import (
	"bytes"
	"strings"

	jujucmd "github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/resource"
	"github.com/juju/juju/resource/cmd"
	coretesting "github.com/juju/juju/testing"
)

var _ = gc.Suite(&ShowSuite{})

type ShowSuite struct {
	testing.IsolationSuite

	stub   *testing.Stub
	client *stubClient
}

func (s *ShowSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.stub = &testing.Stub{}
	s.client = &stubClient{stub: s.stub}
}

func (s *ShowSuite) newAPIClient(c *cmd.ShowCommand) (cmd.ShowAPI, error) {
	s.stub.AddCall("newAPIClient", c)
	if err := s.stub.NextErr(); err != nil {
		return nil, errors.Trace(err)
	}

	return s.client, nil
}

func (s *ShowSuite) TestInfo(c *gc.C) {
	var command cmd.ShowCommand
	info := command.Info()

	c.Check(info, jc.DeepEquals, &jujucmd.Info{
		Name:    "show-resources",
		Args:    "service-id",
		Purpose: "display the charm-defined resources for a service",
		Doc: `
This command will report the resources defined by a charm.

The resources are looked up in the service's charm metadata.
`,
	})
}

func (s *ShowSuite) TestOkay(c *gc.C) {
	specs := cmd.NewSpecs(c,
		"website:.tgz of your website",
		"music:mp3 of your backing vocals",
	)
	s.client.ReturnListSpecs = append(s.client.ReturnListSpecs, resource.SpecsResult{
		Service: "a-service",
		Specs:   specs,
	})

	command := cmd.NewShowCommand(s.newAPIClient)
	code, stdout, stderr := runShow(c, command, "a-service")
	c.Check(code, gc.Equals, 0)

	c.Check(stdout, gc.Equals, `
RESOURCE FROM   REV COMMENT                    
website  upload -   .tgz of your website       
music    upload -   mp3 of your backing vocals 

`[1:])
	c.Check(stderr, gc.Equals, "")
	s.stub.CheckCallNames(c, "newAPIClient", "ListSpecs", "Close")
	s.stub.CheckCall(c, 0, "newAPIClient", command)
	s.stub.CheckCall(c, 1, "ListSpecs", []string{"a-service"})
}

func (s *ShowSuite) TestNoResources(c *gc.C) {
	command := cmd.NewShowCommand(s.newAPIClient)
	code, stdout, stderr := runShow(c, command, "a-service")
	c.Check(code, gc.Equals, 0)

	c.Check(stdout, gc.Equals, `
RESOURCE FROM REV COMMENT 

`[1:])
	c.Check(stderr, gc.Equals, "")
	s.stub.CheckCallNames(c, "newAPIClient", "ListSpecs", "Close")
}

func (s *ShowSuite) TestOutputFormats(c *gc.C) {
	specs := []resource.Spec{
		cmd.NewSpec(c, "website", ".tgz", ".tgz of your website"),
		cmd.NewSpec(c, "music", ".mp3", "mp3 of your backing vocals"),
	}
	s.client.ReturnListSpecs = append(s.client.ReturnListSpecs, resource.SpecsResult{
		Service: "a-service",
		Specs:   specs,
	})

	formats := map[string]string{
		"tabular": `
RESOURCE FROM   REV COMMENT                    
website  upload -   .tgz of your website       
music    upload -   mp3 of your backing vocals 

`[1:],
		"yaml": `
- name: website
  type: file
  path: website.tgz
  comment: .tgz of your website
  origin: upload
- name: music
  type: file
  path: music.mp3
  comment: mp3 of your backing vocals
  origin: upload
`[1:],
		"json": strings.Replace(""+
			"["+
			"  {"+
			`    "name":"website",`+
			`    "type":"file",`+
			`    "path":"website.tgz",`+
			`    "comment":".tgz of your website",`+
			`    "origin":"upload"`+
			"  },{"+
			`    "name":"music",`+
			`    "type":"file",`+
			`    "path":"music.mp3",`+
			`    "comment":"mp3 of your backing vocals",`+
			`    "origin":"upload"`+
			"  }"+
			"]\n",
			"  ", "", -1),
	}
	for format, expected := range formats {
		command := cmd.NewShowCommand(s.newAPIClient)
		args := []string{
			"--format", format,
			"a-service",
		}
		code, stdout, stderr := runShow(c, command, args...)
		c.Check(code, gc.Equals, 0)

		c.Check(stdout, gc.Equals, expected)
		c.Check(stderr, gc.Equals, "")
	}
}

func runShow(c *gc.C, command *cmd.ShowCommand, args ...string) (int, string, string) {
	ctx := coretesting.Context(c)
	code := jujucmd.Main(command, ctx, args)
	stdout := ctx.Stdout.(*bytes.Buffer).Bytes()
	stderr := ctx.Stderr.(*bytes.Buffer).Bytes()
	return code, string(stdout), string(stderr)
}