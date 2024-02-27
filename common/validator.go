package common

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type DirValidator struct{}

func (v DirValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Validate that the path exists")
}

func (v DirValidator) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("Validate that the path exists")
}

func (v DirValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	p := req.ConfigValue.ValueString()
	if len(p) == 0 {
		return
	}
	stat, err := os.Stat(p)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Path",
			fmt.Sprintf("Path %q does not exist: %s", p, err),
		)
		return
	}
	if !stat.IsDir() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Path",
			fmt.Sprintf("Path %q is not a directory", p),
		)
	}
	return
}
