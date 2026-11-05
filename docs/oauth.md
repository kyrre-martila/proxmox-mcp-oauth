# Native OAuth for ChatGPT

OAuth is optional and HTTP-only. Existing MCP_AUTH_TOKEN clients continue to work
alongside it. Never combine OAuth with MCP_ALLOW_UNAUTHENTICATED.

## Configuration

| Variable | Value |
| --- | --- |
| MCP_TRANSPORT | http |
| MCP_OAUTH_ISSUER | Canonical public HTTPS origin, e.g. https://mcp.example.com (no path or trailing slash) |
| MCP_OAUTH_ADMIN_HASH | bcrypt hash of a unique strong administrator password |
| MCP_OAUTH_STATE_DIR | Absolute persistent directory path, mode 0700 |
| MCP_OAUTH_REDIRECT_URIS | Comma-separated exact allowed HTTPS callback URLs |
| MCP_ALLOWED_HOSTS | Include the issuer hostname and existing allowed hosts |

These variables support the existing _FILE convention. Prefer
MCP_OAUTH_ADMIN_HASH_FILE with a mounted secret. Generate a hash using a trusted
local password manager or htpasswd -nBC 12 admin; retain only the portion after
admin:. Use cost 12 or higher. Never put the plaintext password in environment
variables, command arguments or compose files.

Keep existing Proxmox configuration. Write, delete and exec permissions retain
their existing defaults. OAuth scope mcp delegates only currently enabled tools;
it cannot override server controls. All authorized clients use the same configured
Proxmox service identity. This is a single-operator server.

## Separate Docker test deployment

Build the PR image using docker build -t proxmox-mcp:oauth-test .
Test using a separate container, port and environment file, without replacing the
production container. Mount a private directory owned by UID 65532 with mode 0700
at /oauth-state, set MCP_OAUTH_STATE_DIR=/oauth-state, and mount the administrator
hash read-only at /run/secrets/oauth-admin with
MCP_OAUTH_ADMIN_HASH_FILE=/run/secrets/oauth-admin.

Retain --read-only, --cap-drop=ALL and --security-opt=no-new-privileges. Publish a
separate local port such as 127.0.0.1:18080:8080. Supply the existing Proxmox
read-only API credential in the separate env file. Use a separately approved HTTPS
test endpoint forwarding the canonical Host unchanged. OAuth endpoints require
that exact Host (including port if configured); forwarded headers never determine
the issuer. The existing compose file, release publishing and Cloudflare
configuration are unchanged.

## ChatGPT connection

1. Find the exact callback URL in the current ChatGPT custom MCP OAuth flow.
   Add it to MCP_OAUTH_REDIRECT_URIS. A common value is
   https://chatgpt.com/connector_platform/oauth_redirect; verify it in the client.
   Wildcards and unlisted redirects are refused.
2. Add https://mcp.example.com/mcp as a custom MCP server. Choose OAuth and leave
   client credentials blank for public dynamic registration.
3. ChatGPT discovers the server, registers a public client, requests scope mcp
   (or omits scope), and uses Authorization Code with PKCE S256.
4. Check the displayed client and redirect. Enter the administrator password and
   explicitly Authorize, or Deny. There is no public signup or automatic consent.
5. Token exchange must use the same redirect and the verifier. Both authorization
   and token/refresh requests require resource=https://mcp.example.com/mcp.
   MCP calls carry the access token in the Authorization Bearer header.

Discovery endpoints are /.well-known/oauth-protected-resource/mcp (also a root
alias) and /.well-known/oauth-authorization-server. Unauthorized MCP calls return
401 with a resource_metadata Bearer challenge. Supported registration is RFC 7591
dynamic registration, token_endpoint_auth_method=none. Confidential client
authentication and URL client metadata documents are not supported.

A real ChatGPT connection requires a separately approved HTTPS integration test.
Automated tests do not prove compatibility with every current ChatGPT variant.

## Security and lifecycle

Codes expire after two minutes, access tokens after fifteen minutes, and grants
after thirty days absolutely. Refresh rotates both tokens; reuse of an old
refresh token revokes the entire grant. After 1000 rotations, new consent is
required. Tokens are random 256-bit opaque values; only SHA-256 digests persist.
PKCE uses golang.org/x/oauth2 and password verification uses x/crypto/bcrypt.
No custom cryptographic algorithms or JWT signing keys are used.

Revoke using a form POST to /oauth/revoke with client_id and token (access or
refresh). Unknown tokens return 200. Keep these credentials out of logs and shell
history. To revoke everything, stop the process, remove its state.json and restart.
Changing the administrator password hash does not revoke existing grants; reset
state if compromise is suspected. Static tokens are revoked separately by
replacing MCP_AUTH_TOKEN.

Use one process per state directory on a local durable filesystem. A maintained
cross-platform file lock rejects a second process; serialized atomic 0600
snapshots persist code consumption and rotation. Do not share state across
replicas or restore stale backups: this can resurrect revoked credentials.
Backups contain sensitive client/grant metadata; protect and encrypt them at the
storage layer. Persistence errors disable OAuth until operator recovery. Reset
state before restart if revocation durability is uncertain.

Registration is capped at 100 clients (operator reset required if exhausted),
pending consent/codes at 100 each, grants at 1000. Mutations are globally limited
to two requests per second, burst ten, including password attempts. This bounds
work but does not prevent sustained distributed denial of service.

Consent requires a one-time request, a Secure HttpOnly SameSite Strict cookie,
same-origin POST and the correct administrator password. Keep TLS and the state
volume protected. The HTTP listener expects TLS termination in front of it.
Do not log authorization query strings, headers, bodies or token responses at
your ingress. The application never logs OAuth credentials.
