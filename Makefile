host_name=terraform-example.com
namespace=nikolaymatrosov
type=goose
version=0.0.1
target=darwin_arm64

local-publish::
	go build -o terraform-provider-goose

	mkdir -p ~/.terraform.d/plugins/$(host_name)/$(namespace)/$(type)/$(version)/$(target)
	cp terraform-provider-goose ~/.terraform.d/plugins/$(host_name)/$(namespace)/$(type)/$(version)/$(target)/terraform-provider-goose_v$(version)