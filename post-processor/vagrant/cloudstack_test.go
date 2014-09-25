package vagrant

import (
	"testing"
)

func TestCloudStackProvider_impl(t *testing.T) {
	var _ Provider = new(CloudStackProvider)
}

func TestCloudStackProvider_KeepInputArtifact(t *testing.T) {
	p := new(CloudStackProvider)

	if !p.KeepInputArtifact() {
		t.Fatal("should keep input artifact")
	}
}
