---
title: "Static Configuration"
date: 2022-06-09T18:57:50+02:00
lastmod: 2022-06-09T18:57:50+02:00
draft: true
menu:
  docs:
    weight: 10
    parent: "Reference"
---

Below you can find possible contents (not exhaustive) for Heimdall's `config.yaml` file. Head over to configuration documentation to get detailed explanation.

```yaml
serve:
  api:
    host: 127.0.0.1
    port: 4468
    verbose_errors: true
    timeout:
      read: 2s
      write: 5s
      idle: 2m
    cors:
      allowed_origins:
        - example.org
      allowed_methods:
        - GET
        - POST
      allowed_headers:
        - Authorization
      exposed_headers:
        - X-My-Header
      allow_credentials: true
      max_age: 1m
    tls:
      key: /path/to/key/file.pem
      cert: /path/to/cert/file.pem
    trusted_proxies:
      - 192.168.1.0/24

  proxy:
    host: 127.0.0.1
    port: 4469
    verbose_errors: false
    timeout:
      read: 2s
      write: 5s
      idle: 2m
    cors:
      allowed_origins:
        - example.org
      allowed_methods:
        - GET
        - POST
      allowed_headers:
        - Authorization
      exposed_headers:
        - X-My-Header
      allow_credentials: true
      max_age: 1m
    tls:
      key: /path/to/key/file.pem
      cert: /path/to/cert/file.pem
    trusted_proxies:
      - 192.168.1.0/24

log:
  level: debug
  format: text
  
tracing:
  service_name: heimdall
  provider: jaeger
  
metrics:
  prometheus:
    host: 0.0.0.0
    port: 9000
    metrics_path: /metrics

pipeline:
  authenticators:
    - id: "noop_authenticator"
      type: noop
    - id: "anonymous_authenticator"
      type: anonymous
    - id: "unauthorized_authenticator"
      type: unauthorized
    - id: "kratos_session_authenticator"
      type: generic
      config:
        identity_info_endpoint:
          url: http://127.0.0.1:4433/sessions/whoami
          retry:
            max_delay: 300ms
            give_up_after: 2s
        authentication_data_source:
          - cookie: ory_kratos_session
        session:
          subject_attributes_from: "@this"
          subject_id_from: "identity.id"
    - id: "hydra_authenticator"
      type: oauth2_introspection
      config:
        introspection_endpoint:
          url: http://hydra:4445/oauth2/introspect
          retry:
            max_delay: 300ms
            give_up_after: 2s
          auth:
            type: basic_auth
            config:
              user: foo
              password: bar
        assertions:
          issuers:
            - http://127.0.0.1:4444/
          scopes:
            - foo
            - bar
          audience:
            - bla
        session:
          subject_attributes_from: "@this"
          subject_id_from: "sub"
    - id: "jwt_authenticator"
      type: jwt
      config:
        jwks_endpoint:
          url: http://foo/token
          method: GET
        jwt_from:
          - header: Authorization
            strip_prefix: Bearer
        assertions:
          audience:
            - bla
          scopes:
            - foo
          allowed_algorithms:
            - RSA
          issuers:
            - bla
        session:
          subject_attributes_from: "@this"
          subject_id_from: "identity.id"
        cache_ttl: 5m

  authorizers:
    - id: "allow_all_authorizer"
      type: allow
    - id: "deny_all_authorizer"
      type: deny
    - id: "keto_authorizer"
      type: remote
      config:
        endpoint:
          url: http://keto
          method: POST
          headers:
            foo-bar: "{{ .Subject }}"
        payload: "https://bla.bar"
        forward_response_headers_to_upstream:
          - bla-bar
    - id: "attributes_based_authorizer"
      type: local
      config:
        script: "console.log('New JS script')"

  hydrators:
    - id: "subscription_hydrator"
      type: generic
      config:
        endpoint:
          url: http://foo.bar
          method: GET
          headers:
            bla: bla
        payload: http://foo
    - id: "profile_data_hydrator"
      type: generic
      config:
        endpoint:
          url: http://profile
          headers:
            foo: bar

  mutators:
    - id: "jwt"
      type: jwt
      config:
        ttl: 5m
        claims: "{'user': {{ quote .ID }} }"
    - id: "bla"
      type: header
      config:
        headers:
          foo-bar: bla
    - id: "blabla"
      type: cookie
      config:
        cookies:
          foo-bar: '{{ .ID }}'

  error_handlers:
    - id: default
      type: default
    - id: authenticate_with_kratos
      type: redirect
      config:
        to: http://127.0.0.1:4433/self-service/login/browser
        return_to_query_parameter: return_to
        when:
          - error:
            - authentication_error
            - authorization_error
            request_headers:
              Accept:
              - '*/*'

rules:
  default:
    methods:
      - GET
      - POST
    execute:
      - authenticator: anonymous_authenticator
      - mutator: jwt
    on_error:
      - error_handler: authenticate_with_kratos

  providers:
    file:
      src: test_rules.yaml
      watch: true
```
