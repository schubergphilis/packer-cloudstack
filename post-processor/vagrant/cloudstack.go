package vagrant

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"text/template"

	"github.com/mitchellh/packer/packer"
)

type CloudStackProvider struct{}

func (p *CloudStackProvider) KeepInputArtifact() bool {
	return true
}

func (p *CloudStackProvider) Process(ui packer.Ui, artifact packer.Artifact, dir string) (vagrantfile string, metadata map[string]interface{}, err error) {
	// Create the metadata
	metadata = map[string]interface{}{"provider": "cloudstack"}

	// Build up the template data to build our Vagrantfile
	tplData := &cloudStackVagrantfileTemplate{}

	url, err := url.Parse(artifact.Id())
	if err != nil {
		err = fmt.Errorf("Poorly formatted artifact ID: %s", artifact.Id())
		return
	}
	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		err = fmt.Errorf("Network address has an invalid form: %s", artifact.Id())
		return
	}
	tplData.Host = host
	tplData.Port = port
	tplData.Path = url.Path
	tplData.Scheme = url.Scheme
	values := url.Query()
	tplData.TemplateId = values.Get("templateid")

	// Build up the Vagrantfile
	var contents bytes.Buffer
	tpl := template.Must(template.New("vf").Parse(defaultCloudStackVagrantfile))
	err = tpl.Execute(&contents, tplData)
	vagrantfile = contents.String()

	return
}

type cloudStackVagrantfileTemplate struct {
	Host       string
	Path       string
	Port       string
	Scheme     string "http"
	TemplateId string
}

var defaultCloudStackVagrantfile = `
Vagrant.configure("2") do |config|
  config.vm.provider "cloudstack" do |cloudstack|
    cloudstack.host = "{{ .Host }}"
    cloudstack.path = "{{ .Path }}"
    cloudstack.port = "{{ .Port }}"
    cloudstack.scheme = "{{ .Scheme }}"

    cloudstack.template_id = "{{ .TemplateId }}"
  end
end
`
