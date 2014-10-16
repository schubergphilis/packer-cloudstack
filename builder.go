// The cloudstack package contains a packer.Builder implementation
// that builds Cloudstack images (templates).

package cloudstack

import (
	"errors"
	"fmt"
	"github.com/mindjiver/gopherstack"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/packer"
	"log"
	"os"
	"time"
)

// The unique id for the builder
const BuilderId = "mindjiver.cloudstack"

// Configuration tells the builder the credentials to use while
// communicating with Cloudstack and describes the template you are
// creating
type config struct {
	common.PackerConfig `mapstructure:",squash"`

	APIURL    string `mapstructure:"api_url"`
	APIKey    string `mapstructure:"api_key"`
	SecretKey string `mapstructure:"secret_key"`

	RawSSHTimeout   string `mapstructure:"ssh_timeout"`
	RawStateTimeout string `mapstructure:"state_timeout"`

	// Time to wait before issuing the API call to detach the
	// bootstrap/installation ISO from the virtual machine.
	RawDetachISOWait string `mapstructure:"detach_iso_wait"`

	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`

	SSHUsername string `mapstructure:"ssh_username"`
	SSHPort     uint   `mapstructure:"ssh_port"`
	SSHKeyPath  string `mapstructure:"ssh_key_path"`
	SSHPassword string `mapstructure:"ssh_password"`

	// These are unexported since they're set by other fields
	// being set.
	sshTimeout    time.Duration
	stateTimeout  time.Duration
	detachISOWait time.Duration

	HTTPDir     string `mapstructure:"http_directory"`
	HTTPPortMin uint   `mapstructure:"http_port_min"`
	HTTPPortMax uint   `mapstructure:"http_port_max"`

	// Neccessary settings for Cloudstack to be able to spin up
	// Virtual Machine with either template or a ISO.
	ServiceOfferingId string   `mapstructure:"service_offering_id"`
	TemplateId        string   `mapstructure:"template_id"`
	ZoneId            string   `mapstructure:"zone_id"`
	NetworkIds        []string `mapstructure:"network_ids"`
	DiskOfferingId    string   `mapstructure:"disk_offering_id"`
	UserData          string   `mapstructure:"user_data"`
	Hypervisor        string   `mapstructure:"hypervisor"`

	// Tell Cloudstack under which name, description to save the
	// template.
	TemplateName            string `mapstructure:"template_name"`
	TemplateDisplayText     string `mapstructure:"template_display_text"`
	TemplateOSId            string `mapstructure:"template_os_id"`
	TemplateScalable        bool   `mapstructure:"template_scalable"`
	TemplatePublic          bool   `mapstructure:"template_public"`
	TemplateFeatured        bool   `mapstructure:"template_featured"`
	TemplateExtractable     bool   `mapstructure:"template_extractable"`
	TemplatePasswordEnabled bool   `mapstructure:"template_password_enabled"`

	tpl *packer.ConfigTemplate
}

type Builder struct {
	config config
	runner multistep.Runner
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	md, err := common.DecodeConfig(&b.config, raws...)
	if err != nil {
		return nil, err
	}

	b.config.tpl, err = packer.NewConfigTemplate()
	if err != nil {
		return nil, err
	}
	b.config.tpl.UserVars = b.config.PackerUserVars

	// Accumulate any errors
	errs := common.CheckUnusedConfig(md)

	if b.config.APIURL == "" {
		// Default to environment variable for API URL
		b.config.APIURL = os.Getenv("CLOUDSTACK_API_URL")
	}

	if b.config.APIKey == "" {
		// Default to environment variable for API key
		b.config.APIKey = os.Getenv("CLOUDSTACK_API_KEY")
	}

	if b.config.SecretKey == "" {
		// Default to environment variable for API secret
		b.config.SecretKey = os.Getenv("CLOUDSTACK_SECRET_KEY")
	}

	if b.config.HTTPPortMin == 0 {
		b.config.HTTPPortMin = 8000
	}

	if b.config.HTTPPortMax == 0 {
		b.config.HTTPPortMax = 9000
	}

	if b.config.TemplateName == "" {
		// Default to packer-{{ unix timestamp (utc) }}
		b.config.TemplateName = "packer-{{timestamp}}"
	}

	if b.config.TemplateDisplayText == "" {
		b.config.TemplateDisplayText = "Packer_Generated_Template"
	}

	if b.config.TemplateOSId == "" {
		// Default to Other 64 bit OS
		b.config.TemplateOSId = "103"
	}

	if b.config.SSHUsername == "" {
		// Default to "root". You can override this if your
		// source template has a different user account.
		b.config.SSHUsername = "root"
	}

	if b.config.SSHPort == 0 {
		// Default to port 22
		b.config.SSHPort = 22
	}

	if b.config.RawSSHTimeout == "" {
		// Default to 10 minute timeouts
		b.config.RawSSHTimeout = "10m"
	}

	if b.config.RawStateTimeout == "" {
		// Default to 5 minute timeouts waiting for desired
		// state. i.e waiting for virtual machine to become
		// active
		b.config.RawStateTimeout = "5m"
	}

	if b.config.RawDetachISOWait == "" {
		// Default to wait 10 seconds before detaching the ISO
		// from the started virtual machine.
		b.config.RawDetachISOWait = "10s"
	}

	templates := map[string]*string{
		"api_url":               &b.config.APIURL,
		"api_key":               &b.config.APIKey,
		"secret_key":            &b.config.SecretKey,
		"ssh_timeout":           &b.config.RawSSHTimeout,
		"state_timeout":         &b.config.RawStateTimeout,
		"detach_iso_wait":       &b.config.RawDetachISOWait,
		"ssh_username":          &b.config.SSHUsername,
		"ssh_key_path":          &b.config.SSHKeyPath,
		"ssh_password":          &b.config.SSHPassword,
		"http_directory":        &b.config.HTTPDir,
		"service_offering_id":   &b.config.ServiceOfferingId,
		"template_id":           &b.config.TemplateId,
		"zone_id":               &b.config.ZoneId,
		"disk_offering_id":      &b.config.DiskOfferingId,
		"hypervisor":            &b.config.Hypervisor,
		"template_name":         &b.config.TemplateName,
		"template_display_text": &b.config.TemplateDisplayText,
		"template_os_id":        &b.config.TemplateOSId,
	}

	for n, ptr := range templates {
		var err error
		*ptr, err = b.config.tpl.Process(*ptr, nil)
		if err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error processing %s: %s", n, err))
		}
	}

	validates := map[string]*string{
		"user_data": &b.config.UserData,
	}

	for n, ptr := range validates {
		if err := b.config.tpl.Validate(*ptr); err != nil {
			errs = packer.MultiErrorAppend(
				errs, fmt.Errorf("Error parsing %s: %s", n, err))
		}
	}

	if b.config.HTTPPortMin > b.config.HTTPPortMax {
		errs = packer.MultiErrorAppend(
			errs, errors.New("http_port_min must be less than http_port_max"))
	}

	// Required configurations that will display errors if not set
	if b.config.APIURL == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("CLOUDSTACK_API_URL in env (APIURL in json) must be specified"))
	}

	if b.config.APIKey == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("CLOUDSTACK_API_KEY in env (APIKey in json) must be specified"))
	}

	if b.config.SecretKey == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("CLOUDSTACK_SECRET_KEY in env (SecretKey in json) must be specified"))
	}

	if b.config.ServiceOfferingId == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("service_offering_id must be specified"))
	}

	if b.config.TemplateId == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("template_id must be specified"))
	}

	if b.config.ZoneId == "" {
		errs = packer.MultiErrorAppend(
			errs, errors.New("zone_id must be specified"))
	}

	sshTimeout, err := time.ParseDuration(b.config.RawSSHTimeout)
	if err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Failed parsing ssh_timeout: %s", err))
	}
	b.config.sshTimeout = sshTimeout

	detachISOWait, err := time.ParseDuration(b.config.RawDetachISOWait)
	if err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Failed parsing iso_detach_wait: %s", err))
	}
	b.config.detachISOWait = detachISOWait

	stateTimeout, err := time.ParseDuration(b.config.RawStateTimeout)
	if err != nil {
		errs = packer.MultiErrorAppend(
			errs, fmt.Errorf("Failed parsing state_timeout: %s", err))
	}
	b.config.stateTimeout = stateTimeout

	if errs != nil && len(errs.Errors) > 0 {
		return nil, errs
	}

	common.ScrubConfig(b.config, b.config.APIKey, b.config.SecretKey)
	return nil, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {
	// Initialize the Cloudstack API client
	client := gopherstack.CloudstackClient{}.New(b.config.APIURL, b.config.APIKey,
		b.config.SecretKey, b.config.InsecureSkipVerify)

	// Set up the state
	state := new(multistep.BasicStateBag)
	state.Put("config", b.config)
	state.Put("client", client)
	state.Put("hook", hook)
	state.Put("ui", ui)

	// Build the steps
	steps := []multistep.Step{
		new(stepHTTPServer),
		new(stepCreateSSHKeyPair),
		new(stepDeployVirtualMachine),
		new(stepVirtualMachineState),
		new(stepDetachIso),
		&common.StepConnectSSH{
			SSHAddress:     sshAddress,
			SSHConfig:      sshConfig,
			SSHWaitTimeout: b.config.sshTimeout,
		},
		new(common.StepProvision),
		new(stepStopVirtualMachine),
		new(stepCreateTemplate),
	}

	// Run the steps
	if b.config.PackerDebug {
		b.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		b.runner = &multistep.BasicRunner{Steps: steps}
	}

	b.runner.Run(state)

	// If there was an error, return that
	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	// If we were interrupted or cancelled, then just exit.
	if _, ok := state.GetOk(multistep.StateCancelled); ok {
		return nil, errors.New("Build was cancelled.")
	}

	if _, ok := state.GetOk(multistep.StateHalted); ok {
		return nil, errors.New("Build was halted.")
	}

	if _, ok := state.GetOk("template_name"); !ok {
		log.Println("Failed to find template_name in state. Bug?")
		return nil, nil
	}

	artifact := &Artifact{
		templateName: state.Get("template_name").(string),
		templateId:   state.Get("template_id").(string),
		client:       client,
	}

	return artifact, nil
}

func (b *Builder) Cancel() {
	if b.runner != nil {
		log.Println("Cancelling the step runner...")
		b.runner.Cancel()
	}
}
