package plugin

import (
	"fmt"
	"net/url"
	"strings"

	"kandev-plugin-bitbucket/internal/auth"
	"kandev-plugin-bitbucket/internal/cloud"
	"kandev-plugin-bitbucket/internal/datacenter"
	"kandev-plugin-bitbucket/internal/domain"
)

func validateConnection(settings ConnectionSettings) error {
	if err := validateConnectionAuthentication(settings); err != nil {
		return err
	}
	switch settings.Product {
	case domain.ProductCloud:
		if settings.CloudWorkspace == "" {
			return fmt.Errorf("Bitbucket Cloud workspace is required")
		}
		if settings.BaseURL != "" {
			return fmt.Errorf("Bitbucket Cloud API endpoint is fixed")
		}
	case domain.ProductDataCenter:
		if settings.BaseURL == "" {
			return fmt.Errorf("Bitbucket Data Center URL is required")
		}
		if _, err := datacenter.NewConnection(datacenter.ConnectionOptions{BaseURL: settings.BaseURL}); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Bitbucket product must be cloud or data_center")
	}
	return nil
}

func normalizedAuthMethod(method string) string {
	return strings.ToLower(strings.TrimSpace(method))
}

func oauthScopes(product domain.Product) []string {
	if product == domain.ProductDataCenter {
		return []string{"REPO_READ", "REPO_WRITE"}
	}
	return []string{"account", "repository", "repository:write", "pullrequest", "pullrequest:write"}
}

func validateConnectionAuthentication(settings ConnectionSettings) error {
	switch normalizedAuthMethod(settings.AuthMethod) {
	case "":
		return fmt.Errorf("Bitbucket authentication method is required")
	case "oauth":
		if settings.OAuthGeneration == 0 || settings.OAuthClientID == "" || settings.OAuthRedirectURL == "" {
			return fmt.Errorf("Bitbucket OAuth client registration is required")
		}
		if settings.Product == domain.ProductDataCenter && strings.TrimSpace(settings.AuthIdentity) == "" {
			return fmt.Errorf("Bitbucket Data Center OAuth Git username is required")
		}
		return nil
	case "api_token":
		if settings.Product != domain.ProductCloud {
			return fmt.Errorf("Bitbucket Data Center authentication method must be user_pat, project_token, repository_token, or oauth")
		}
		if strings.TrimSpace(settings.AuthIdentity) == "" {
			return fmt.Errorf("Bitbucket Cloud API-token email is required")
		}
		return nil
	case "user_pat":
		if settings.Product != domain.ProductDataCenter {
			return fmt.Errorf("Bitbucket Cloud authentication method must be api_token or oauth")
		}
		if strings.TrimSpace(settings.AuthIdentity) == "" {
			return fmt.Errorf("Bitbucket Data Center user PAT username is required")
		}
		return nil
	case "project_token", "repository_token":
		if settings.Product != domain.ProductDataCenter {
			return fmt.Errorf("Bitbucket Cloud authentication method must be api_token or oauth")
		}
		return nil
	default:
		return fmt.Errorf("unsupported Bitbucket authentication method")
	}
}

func cloudAuthentication(settings ConnectionSettings) (cloud.Authentication, error) {
	switch normalizedAuthMethod(settings.AuthMethod) {
	case "oauth":
		return cloud.Authentication{Mode: cloud.AuthenticationOAuth}, nil
	case "api_token":
		return cloud.Authentication{Mode: cloud.AuthenticationAPIToken, Email: settings.AuthIdentity}, nil
	default:
		return cloud.Authentication{}, fmt.Errorf("unsupported Bitbucket Cloud authentication method")
	}
}

func dataCenterAuthentication(settings ConnectionSettings) (datacenter.Authentication, error) {
	switch normalizedAuthMethod(settings.AuthMethod) {
	case "oauth":
		return datacenter.Authentication{Mode: datacenter.AuthenticationOAuth, Username: settings.AuthIdentity}, nil
	case "user_pat":
		return datacenter.Authentication{Mode: datacenter.AuthenticationPAT, Username: settings.AuthIdentity}, nil
	case "project_token":
		return datacenter.Authentication{Mode: datacenter.AuthenticationProjectToken}, nil
	case "repository_token":
		return datacenter.Authentication{Mode: datacenter.AuthenticationRepositoryToken}, nil
	default:
		return datacenter.Authentication{}, fmt.Errorf("unsupported Bitbucket Data Center authentication method")
	}
}

func validateHTTPSURL(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must be a credential-free HTTPS URL", label)
	}
	return nil
}

func validateOAuthRedirectURL(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s must be a credential-free URL without a query or fragment", label)
	}
	if err := auth.ValidateOAuthRedirectURL(parsed); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	return nil
}
