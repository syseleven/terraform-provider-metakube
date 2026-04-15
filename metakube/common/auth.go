package common

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	fwdiag "github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/mitchellh/go-homedir"
)

type AuthError struct {
	Message    string
	Attributes []string
}

func (e *AuthError) Error() string {
	return e.Message
}

func NewAuthError(message string, attributes ...string) *AuthError {
	return &AuthError{
		Message:    message,
		Attributes: attributes,
	}
}

func NewAuth(token, tokenPath, terraformVersion string) (runtime.ClientAuthInfoWriter, error) {
	resolvedToken, err := resolveToken(token, tokenPath)
	if err != nil {
		return nil, err
	}

	auth := runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
		if err := r.SetHeaderParam("Authorization", "Bearer "+resolvedToken); err != nil {
			return err
		}
		return r.SetHeaderParam("User-Agent", fmt.Sprintf("Terraform/%s", terraformVersion))
	})

	return auth, nil
}

func resolveToken(token, tokenPath string) (string, error) {
	if token != "" {
		return token, nil
	}

	if tokenPath != "" {
		expandedPath, err := homedir.Expand(tokenPath)
		if err != nil {
			return "", NewAuthError(
				fmt.Sprintf("failed to expand token path: %v", err),
				"token_path",
			)
		}

		rawToken, err := os.ReadFile(expandedPath)
		if err != nil {
			return "", NewAuthError(
				fmt.Sprintf("failed to read token file: %v", err),
				"token_path",
			)
		}

		return string(bytes.Trim(rawToken, "\n")), nil
	}

	return "", NewAuthError(
		"missing authorization token: provide either 'token' or 'token_path'",
		"token", "token_path",
	)
}

func ToFrameworkDiagnostics(err error) fwdiag.Diagnostics {
	if err == nil {
		return nil
	}

	var diags fwdiag.Diagnostics
	var authErr *AuthError
	if errors.As(err, &authErr) {
		for _, attr := range authErr.Attributes {
			diags.AddAttributeError(
				path.Root(attr),
				"Authentication Error",
				authErr.Message,
			)
		}

		if len(authErr.Attributes) == 0 {
			diags.AddError("Authentication Error", authErr.Message)
		}
	} else {
		diags.AddError("Authentication Error", err.Error())
	}

	return diags
}
