package cloudstack

import (
	"fmt"
	"github.com/mindjiver/gopherstack"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

type stepCreateTags struct{}

func (s *stepCreateTags) Run(state multistep.StateBag) multistep.StepAction {
	client := state.Get("client").(*gopherstack.CloudstackClient)
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(config)
	template := state.Get("template_id").(string)

	if len(c.TemplateTags) > 0 {
		ui.Say(fmt.Sprintf("Adding tags to template (%s)...", template))

		var templateTags []gopherstack.TagArg
		for key, value := range c.TemplateTags {
			ui.Message(fmt.Sprintf("Adding tag: \"%s\": \"%s\"", key, value))
			templateTags = append(templateTags, gopherstack.TagArg{key, value})
		}

		createOpts := &gopherstack.CreateTags{
			Resourceids:  []string{template},
			Resourcetype: "Template",
			Tags:         templateTags,
		}
		_, err := client.CreateTags(createOpts)
		if err != nil {
			err := fmt.Errorf("Error creating tags: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	return multistep.ActionContinue
}

func (s *stepCreateTags) Cleanup(state multistep.StateBag) {
	// No cleanup...
}
