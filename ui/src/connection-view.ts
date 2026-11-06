import {
  connectionIdentity,
  connectionOAuthRegistration,
  connectionSaveBody,
  connectionState,
  deriveOAuthCallbackURL,
  disconnectConnectionInput,
  errorMessage,
  oauthStartInput,
  validateCloudWorkspace,
  validateConnectionIdentity,
  validateOAuthRegistration,
  type ConnectionState,
} from "./view-models";
import { type PluginHost } from "./host-contract";
import {
  Badge,
  isAbortError,
  record,
  text,
  useAbortableOperation,
  usePluginQuery,
} from "./ui-runtime";
import { action } from "./actions";
import { DisconnectConfirmation } from "./disconnect-confirmation";
import { usePluginTranslation } from "./i18n";

export function ConnectionHealth({
  host,
  workspaceId: scopedWorkspaceId,
}: {
  host: PluginHost;
  workspaceId?: string;
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const responsive = host.useResponsiveBreakpoint();
  const connection = usePluginQuery<Record<string, unknown>>(
    host,
    action.connectionGet,
    scopedWorkspaceId ? { workspaceId: scopedWorkspaceId } : undefined,
    Boolean(scopedWorkspaceId),
  );
  const [saving, setSaving] = React.useState(false);
  const [message, setMessage] = React.useState<string | null>(null);
  const [product, setProduct] = React.useState("cloud");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [cloudWorkspace, setCloudWorkspace] = React.useState("");
  const [authMethod, setAuthMethod] = React.useState("api_token");
  const [token, setToken] = React.useState("");
  const details = record(connection.data);
  const [identity, setIdentity] = React.useState("");
  const oauthRegistration = connectionOAuthRegistration(details);
  const [oauthClientId, setOAuthClientId] = React.useState("");
  const [oauthClientSecret, setOAuthClientSecret] = React.useState("");
  const oauthCallbackUrl = deriveOAuthCallbackURL(
    host.api.baseUrl,
    window.location.origin,
  );
  const [disconnectOpen, setDisconnectOpen] = React.useState(false);
  const mutation = useAbortableOperation(host);
  React.useEffect(() => {
    mutation.cancel();
    setSaving(false);
    setMessage(null);
    setProduct("cloud");
    setBaseUrl("");
    setCloudWorkspace("");
    setAuthMethod("api_token");
    setToken("");
    setIdentity("");
    setOAuthClientId("");
    setOAuthClientSecret("");
    setDisconnectOpen(false);
  }, [scopedWorkspaceId]);
  React.useEffect(() => {
    if (!connection.data) return;
    setProduct(text(details.product, "cloud"));
    setBaseUrl(text(details.base_url));
    setCloudWorkspace(text(details.cloud_workspace));
    setAuthMethod(text(details.auth_method, "api_token"));
    setIdentity(text(details.auth_identity));
  }, [connection.data]);
  const state = connectionState(details);
  const identityField = connectionIdentity(product, authMethod, t);
  const formInput = () => ({
    product,
    baseUrl,
    cloudWorkspace,
    authMethod,
    token,
    identity,
    oauthRegistrationConfigured: oauthRegistration.configured,
    oauthClientId,
    oauthClientSecret,
    oauthCallbackUrl,
  });
  const connectionValidationError = () =>
    validateCloudWorkspace(formInput(), t) ??
    validateConnectionIdentity(formInput(), t) ??
    validateOAuthRegistration(formInput(), t);
  const oauthReady = authMethod === "oauth" && !connectionValidationError();
  const label: Record<ConnectionState, string> = {
    unconfigured: t("notConfigured"),
    checking: t("checkingConnection"),
    connected: t("connected"),
    auth_required: t("authenticationRequired"),
    unavailable: t("unavailable"),
  };
  const productDescription =
    details.product === "cloud"
      ? t("bitbucketCloud")
      : details.product === "data_center"
        ? t("bitbucketDataCenter")
        : t("connectDescription");
  const saveConnection = async () => {
    if (!scopedWorkspaceId) return;
    const validationError = connectionValidationError();
    if (validationError) {
      setMessage(validationError);
      return;
    }
    setSaving(true);
    setMessage(null);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.connectionSave,
        {
          workspaceId: scopedWorkspaceId,
          body: {
            ...connectionSaveBody(formInput()),
            probe: true,
          },
        },
        { signal: request.signal },
      );
      if (!request.isCurrent()) return;
      setToken("");
      setOAuthClientSecret("");
      setMessage(t("connectionSaved"));
      connection.refresh();
    } catch (error) {
      if (request.isCurrent() && !isAbortError(error))
        setMessage(errorMessage(error, t));
    } finally {
      if (request.finish()) setSaving(false);
    }
  };
  const startOauth = async () => {
    if (!scopedWorkspaceId) return;
    const validationError = connectionValidationError();
    if (validationError) {
      setMessage(validationError);
      return;
    }
    setSaving(true);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.connectionSave,
        {
          workspaceId: scopedWorkspaceId,
          body: {
            ...connectionSaveBody(formInput()),
            probe: false,
          },
        },
        { signal: request.signal },
      );
      if (!request.isCurrent()) return;
      setOAuthClientSecret("");
      const result = await host.api.invokeAction<Record<string, unknown>>(
        action.oauthStart,
        oauthStartInput(scopedWorkspaceId),
        { signal: request.signal },
      );
      if (!request.isCurrent()) return;
      const href =
        text(record(result).url) || text(record(result).authorization_url);
      if (href) window.location.assign(href);
      else setMessage(t("oauthReady"));
    } catch (error) {
      if (request.isCurrent() && !isAbortError(error))
        setMessage(errorMessage(error, t));
    } finally {
      if (request.finish()) setSaving(false);
    }
  };
  const completeDisconnect = () => {
    setToken("");
    setOAuthClientSecret("");
    setProduct("cloud");
    setBaseUrl("");
    setCloudWorkspace("");
    setAuthMethod("api_token");
    setIdentity("");
    setOAuthClientId("");
    setDisconnectOpen(false);
    setMessage(t("disconnected"));
    connection.refresh();
  };
  const openDisconnectConfirmation = () => {
    if (!scopedWorkspaceId) return;
    if (responsive.isMobile) {
      setDisconnectOpen(true);
      return;
    }
    let modal: { close(): void } | undefined;
    modal = host.openModal({
      title: t("disconnectBitbucket"),
      size: "sm",
      content: () =>
        h(DisconnectConfirmation, {
          host,
          workspaceId: scopedWorkspaceId,
          onSuccess: () => {
            completeDisconnect();
            modal?.close();
          },
          onCancel: () => modal?.close(),
        }),
    });
  };
  return h(
    ui.Card,
    {
      className: "bb-connection",
      "data-testid": "bitbucket-connection-health",
    },
    h(
      ui.CardHeader,
      null,
      h(
        "div",
        { className: "bb-title-row" },
        h(ui.CardTitle, null, t("connection")),
        Badge(
          host,
          label[state],
          state === "connected"
            ? "success"
            : state === "auth_required"
              ? "warning"
              : "neutral",
        ),
      ),
      h(ui.CardDescription, null, productDescription),
    ),
    h(
      ui.CardContent,
      { className: "bb-settings-form" },
      connection.loading
        ? h(ui.Spinner, { "aria-label": t("checkingBitbucketConnection") })
        : null,
      connection.error
        ? h("p", { className: "bb-error", role: "alert" }, connection.error)
        : null,
      message
        ? h("p", { className: "bb-message", role: "status" }, message)
        : null,
      h(ui.Label, { htmlFor: "bitbucket-product" }, t("bitbucketProduct")),
      h(
        ui.Select,
        {
          value: product,
          onValueChange: (next: string) => {
            setProduct(next);
            setAuthMethod(next === "cloud" ? "api_token" : "user_pat");
            setCloudWorkspace("");
            setIdentity("");
            setToken("");
            setOAuthClientId("");
            setOAuthClientSecret("");
          },
        },
        h(
          ui.SelectTrigger,
          { id: "bitbucket-product", className: "min-h-11" },
          h(ui.SelectValue, null),
        ),
        h(
          ui.SelectContent,
          null,
          h(ui.SelectItem, { value: "cloud" }, t("bitbucketCloud")),
          h(ui.SelectItem, { value: "data_center" }, t("bitbucketDataCenter")),
        ),
      ),
      product === "cloud"
        ? h(
            "div",
            { className: "bb-field" },
            h(
              ui.Label,
              { htmlFor: "bitbucket-cloud-workspace" },
              t("bitbucketWorkspace"),
            ),
            h(ui.Input, {
              id: "bitbucket-cloud-workspace",
              "data-testid": "bitbucket-cloud-workspace",
              className: "min-h-11",
              autoComplete: "organization",
              value: cloudWorkspace,
              onChange: (event: { target: { value: string } }) =>
                setCloudWorkspace(event.target.value),
              placeholder: "workspace-slug",
              "aria-describedby": "bitbucket-cloud-workspace-help",
            }),
            h(
              "p",
              {
                id: "bitbucket-cloud-workspace-help",
                className: "bb-capability-note",
              },
              t("workspaceHelp"),
            ),
          )
        : null,
      product === "data_center"
        ? h(
            "div",
            { className: "bb-field" },
            h(ui.Label, { htmlFor: "bitbucket-base-url" }, t("dataCenterUrl")),
            h(ui.Input, {
              id: "bitbucket-base-url",
              className: "min-h-11",
              value: baseUrl,
              onChange: (event: { target: { value: string } }) =>
                setBaseUrl(event.target.value),
              placeholder: "https://bitbucket.example.com/bitbucket",
            }),
          )
        : null,
      h(ui.Label, { htmlFor: "bitbucket-auth-method" }, t("authentication")),
      h(
        ui.Select,
        {
          value: authMethod,
          onValueChange: (next: string) => {
            setAuthMethod(next);
            if (next === "oauth") setToken("");
            if (next !== "oauth") {
              setOAuthClientId("");
              setOAuthClientSecret("");
            }
          },
        },
        h(
          ui.SelectTrigger,
          { id: "bitbucket-auth-method", className: "min-h-11" },
          h(ui.SelectValue, null),
        ),
        h(
          ui.SelectContent,
          null,
          product === "cloud"
            ? [
                h(ui.SelectItem, { value: "api_token" }, t("apiToken")),
                h(ui.SelectItem, { value: "access_token" }, t("cloudAccessToken")),
                h(ui.SelectItem, { value: "oauth" }, t("oauth")),
              ]
            : [
                h(
                  ui.SelectItem,
                  { value: "user_pat" },
                  t("personalAccessToken"),
                ),
                h(
                  ui.SelectItem,
                  { value: "project_token" },
                  t("projectAccessToken"),
                ),
                h(
                  ui.SelectItem,
                  { value: "repository_token" },
                  t("repositoryAccessToken"),
                ),
                h(ui.SelectItem, { value: "oauth" }, t("oauth")),
              ],
        ),
      ),
      authMethod !== "oauth"
        ? h(
            "div",
            { className: "bb-field" },
            h(ui.Label, { htmlFor: "bitbucket-token" }, t("accessToken")),
            h(ui.Input, {
              id: "bitbucket-token",
              type: "password",
              className: "min-h-11",
              autoComplete: "off",
              value: token,
              onChange: (event: { target: { value: string } }) =>
                setToken(event.target.value),
              placeholder: t("tokenPlaceholder"),
            }),
          )
        : null,
      identityField
        ? h(
            "div",
            { className: "bb-field" },
            h(
              ui.Label,
              { htmlFor: "bitbucket-connection-identity" },
              identityField.label,
            ),
            h(ui.Input, {
              id: "bitbucket-connection-identity",
              "data-testid": "bitbucket-connection-identity",
              type: identityField.inputType,
              className: "min-h-11",
              autoComplete:
                identityField.inputType === "email" ? "email" : "username",
              value: identity,
              onChange: (event: { target: { value: string } }) =>
                setIdentity(event.target.value),
              "aria-describedby": "bitbucket-connection-identity-help",
            }),
            h(
              "p",
              {
                id: "bitbucket-connection-identity-help",
                className: "bb-capability-note",
              },
              identityField.help,
            ),
          )
        : null,
      authMethod === "oauth"
        ? h(
            "section",
            {
              className: "bb-oauth-registration",
              "aria-label": t("oauthClientRegistration"),
            },
            oauthRegistration.configured
              ? h(
                  "p",
                  { className: "bb-capability-note", role: "status" },
                  t("oauthRegistrationConfigured"),
                )
              : null,
            h(
              "div",
              { className: "bb-field" },
              h(
                ui.Label,
                { htmlFor: "bitbucket-oauth-client-id" },
                oauthRegistration.configured
                  ? t("oauthClientIdReplace")
                  : t("oauthClientId"),
              ),
              h(ui.Input, {
                id: "bitbucket-oauth-client-id",
                "data-testid": "bitbucket-oauth-client-id",
                className: "min-h-11",
                autoComplete: "off",
                value: oauthClientId,
                onChange: (event: { target: { value: string } }) =>
                  setOAuthClientId(event.target.value),
              }),
            ),
            h(
              "div",
              { className: "bb-field" },
              h(
                ui.Label,
                { htmlFor: "bitbucket-oauth-client-secret" },
                oauthRegistration.configured
                  ? t("oauthClientSecretReplace")
                  : t("oauthClientSecret"),
              ),
              h(ui.Input, {
                id: "bitbucket-oauth-client-secret",
                "data-testid": "bitbucket-oauth-client-secret",
                type: "password",
                className: "min-h-11",
                autoComplete: "off",
                value: oauthClientSecret,
                onChange: (event: { target: { value: string } }) =>
                  setOAuthClientSecret(event.target.value),
                "aria-describedby": "bitbucket-oauth-client-secret-help",
              }),
              h(
                "p",
                {
                  id: "bitbucket-oauth-client-secret-help",
                  className: "bb-capability-note",
                },
                t("oauthSecretHelp"),
              ),
            ),
            h(
              "div",
              { className: "bb-field" },
              h(
                ui.Label,
                { htmlFor: "bitbucket-oauth-callback-url" },
                t("oauthCallbackUrl"),
              ),
              h(ui.Input, {
                id: "bitbucket-oauth-callback-url",
                "data-testid": "bitbucket-oauth-callback-url",
                type: "url",
                className: "min-h-11",
                readOnly: true,
                value: oauthCallbackUrl,
                "aria-describedby": "bitbucket-oauth-callback-url-help",
              }),
              h(
                "p",
                {
                  id: "bitbucket-oauth-callback-url-help",
                  className: "bb-capability-note",
                },
                t("oauthCallbackHelp"),
              ),
            ),
          )
        : null,
      authMethod === "oauth"
        ? h(
            ui.Button,
            {
              type: "button",
              className: "min-h-11",
              disabled: saving || !oauthReady,
              onClick: startOauth,
            },
            t("connectWithOauth"),
          )
        : null,
      h(ui.Separator, null),
      h(
        "div",
        { className: "bb-settings-actions" },
        h(
          ui.Button,
          {
            type: "button",
            variant: "outline",
            className: "min-h-11",
            disabled: saving,
            onClick: saveConnection,
          },
          t("checkConnection"),
        ),
        h(
          ui.Button,
          {
            type: "button",
            variant: "destructive",
            className: "bb-settings-disconnect min-h-11",
            disabled: saving || !scopedWorkspaceId,
            onClick: openDisconnectConfirmation,
          },
          t("disconnectBitbucket"),
        ),
      ),
      responsive.isMobile && scopedWorkspaceId
        ? h(
            ui.Drawer,
            {
              open: disconnectOpen,
              onOpenChange: (open: boolean) => setDisconnectOpen(open),
            },
            h(
              ui.DrawerContent,
              { className: "bb-disconnect-drawer" },
              h(
                ui.DrawerHeader,
                null,
                h(ui.DrawerTitle, null, t("disconnectBitbucket")),
                h(
                  ui.DrawerDescription,
                  null,
                  t("disconnectWorkspaceDescription"),
                ),
              ),
              h(
                "div",
                { className: "bb-drawer-scroll" },
                h(DisconnectConfirmation, {
                  host,
                  workspaceId: scopedWorkspaceId,
                  onSuccess: completeDisconnect,
                  onCancel: () => setDisconnectOpen(false),
                }),
              ),
            ),
          )
        : null,
    ),
  );
}
