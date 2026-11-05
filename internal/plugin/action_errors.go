package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/watches"

	"github.com/kdlbs/kandev/pluginsdk"
)

func categorizedActionError(code pluginsdk.ActionErrorCode, format string, args ...any) error {
	return pluginsdk.CategorizeActionError(code, fmt.Errorf(format, args...))
}

func invalidActionError(format string, args ...any) error {
	return categorizedActionError(pluginsdk.ActionErrorInvalidArgument, format, args...)
}

func forbiddenActionError(format string, args ...any) error {
	return categorizedActionError(pluginsdk.ActionErrorPermissionDenied, format, args...)
}

func notFoundActionError(format string, args ...any) error {
	return categorizedActionError(pluginsdk.ActionErrorNotFound, format, args...)
}

func conflictActionError(format string, args ...any) error {
	return categorizedActionError(pluginsdk.ActionErrorConflict, format, args...)
}

func unavailableActionError(format string, args ...any) error {
	return categorizedActionError(pluginsdk.ActionErrorUnavailable, format, args...)
}

func actionFailureResponse(err error) (*pluginsdk.PluginActionResponse, error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	status, code, message, retryAfter := classifyActionFailure(err)
	body, _ := json.Marshal(map[string]string{"error": message, "code": code})
	headers := map[string]string{"Content-Type": "application/json"}
	if retryAfter != "" {
		headers["Retry-After"] = retryAfter
	}
	return &pluginsdk.PluginActionResponse{Status: status, Headers: headers, Body: body}, nil
}

func categorizeWatchActionError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, watches.ErrWatchNotFound):
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorNotFound, err)
	case errors.Is(err, watches.ErrInvalidWatch):
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorInvalidArgument, err)
	case errors.Is(err, watches.ErrWatchExists):
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorConflict, err)
	case errors.Is(err, watches.ErrWatchPaused):
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorConflict, err)
	case errors.Is(err, watches.ErrConnectionChanged):
		return pluginsdk.CategorizeActionError(pluginsdk.ActionErrorConflict, err)
	default:
		return err
	}
}

func classifyActionFailure(err error) (status int, code, message, retryAfter string) {
	var categorized *pluginsdk.ActionError
	if errors.As(err, &categorized) {
		status = pluginsdk.ActionErrorHTTPStatus(categorized.Code)
		return actionFailureMetadata(categorized.Code, status, categorized.RetryAfter)
	}
	var upstream *domain.ProviderHTTPError
	if errors.As(err, &upstream) {
		code := providerActionErrorCode(upstream.Status)
		return actionFailureMetadata(code, pluginsdk.ActionErrorHTTPStatus(code), upstream.RetryAfter)
	}
	return actionFailureMetadata(pluginsdk.ActionErrorUpstream, http.StatusBadGateway, "")
}

func providerActionErrorCode(status int) pluginsdk.ActionErrorCode {
	switch status {
	case http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusUnprocessableEntity:
		return pluginsdk.ActionErrorInvalidArgument
	case http.StatusUnauthorized:
		return pluginsdk.ActionErrorUnauthenticated
	case http.StatusForbidden:
		return pluginsdk.ActionErrorPermissionDenied
	case http.StatusNotFound:
		return pluginsdk.ActionErrorNotFound
	case http.StatusConflict, http.StatusPreconditionFailed:
		return pluginsdk.ActionErrorConflict
	case http.StatusTooManyRequests:
		return pluginsdk.ActionErrorRateLimited
	case http.StatusRequestTimeout, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return pluginsdk.ActionErrorUnavailable
	default:
		return pluginsdk.ActionErrorUpstream
	}
}

func actionFailureMetadata(
	code pluginsdk.ActionErrorCode,
	status int,
	retryAfter string,
) (int, string, string, string) {
	message := "Bitbucket request failed."
	switch code {
	case pluginsdk.ActionErrorInvalidArgument:
		message = "Invalid Bitbucket action request."
	case pluginsdk.ActionErrorUnauthenticated, pluginsdk.ActionErrorPermissionDenied:
		message = "Bitbucket authorization failed."
	case pluginsdk.ActionErrorNotFound:
		message = "Bitbucket resource was not found."
	case pluginsdk.ActionErrorConflict:
		message = "Bitbucket resource changed. Refresh and retry."
	case pluginsdk.ActionErrorRateLimited:
		message = "Bitbucket rate limit reached. Retry later."
	case pluginsdk.ActionErrorUnavailable:
		message = "Bitbucket connection is unavailable."
	}
	return status, string(code), message, retryAfter
}
